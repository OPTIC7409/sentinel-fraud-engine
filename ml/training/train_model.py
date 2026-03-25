#!/usr/bin/env python3
"""
Train fraud detection model using synthetic transaction data.

Logistic Regression model (v1) for explainability and speed.
Trains on 5 engineered features to predict fraud probability.
"""

import pandas as pd
import numpy as np
from sklearn.model_selection import train_test_split
from sklearn.linear_model import LogisticRegression
from sklearn.metrics import (
    classification_report, 
    confusion_matrix, 
    roc_auc_score, 
    precision_recall_curve,
    roc_curve
)
import joblib
import json
import os
from datetime import datetime

# Model configuration
MODEL_VERSION = "v1.0.0"
TEST_SIZE = 0.2
RANDOM_STATE = 42


def load_data():
    """Load pre-engineered features from CSV."""
    script_dir = os.path.dirname(os.path.abspath(__file__))
    data_path = os.path.join(script_dir, "data", "features.csv")
    
    print(f"Loading data from {data_path}...")
    df = pd.read_csv(data_path)
    
    print(f"Loaded {len(df)} transactions")
    print(f"Fraud rate: {df['is_fraud'].mean() * 100:.2f}%")
    
    return df


def prepare_training_data(df):
    """Split data into train/test sets."""
    # Feature columns (exclude transaction_id and target)
    feature_cols = [
        'amount_normalized',
        'velocity_score',
        'location_deviation',
        'time_anomaly',
        'merchant_category_risk'
    ]
    
    X = df[feature_cols].values
    y = df['is_fraud'].values
    
    # Stratified split to maintain fraud rate in train/test
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, 
        test_size=TEST_SIZE, 
        random_state=RANDOM_STATE,
        stratify=y
    )
    
    print(f"\nTrain set: {len(X_train)} samples ({y_train.mean() * 100:.2f}% fraud)")
    print(f"Test set: {len(X_test)} samples ({y_test.mean() * 100:.2f}% fraud)")
    
    return X_train, X_test, y_train, y_test, feature_cols


def train_model(X_train, y_train):
    """Train Logistic Regression model."""
    print("\nTraining Logistic Regression model...")
    
    # Logistic Regression with balanced class weights
    # (compensates for imbalanced fraud/legit ratio)
    model = LogisticRegression(
        class_weight='balanced',  # Handle class imbalance
        random_state=RANDOM_STATE,
        max_iter=1000,
        solver='lbfgs',
        C=1.0  # Regularization strength
    )
    
    model.fit(X_train, y_train)
    
    print("✓ Model trained successfully")
    
    return model


def evaluate_model(model, X_test, y_test, feature_names):
    """Comprehensive model evaluation."""
    print("\n" + "=" * 60)
    print("MODEL EVALUATION")
    print("=" * 60)
    
    # Predictions
    y_pred = model.predict(X_test)
    y_pred_proba = model.predict_proba(X_test)[:, 1]  # Probability of fraud
    
    # Classification report
    print("\nClassification Report:")
    print(classification_report(y_test, y_pred, target_names=['Legitimate', 'Fraud']))
    
    # Confusion matrix
    cm = confusion_matrix(y_test, y_pred)
    print("\nConfusion Matrix:")
    print(f"                 Predicted Legit  Predicted Fraud")
    print(f"Actually Legit   {cm[0][0]:15d}  {cm[0][1]:15d}")
    print(f"Actually Fraud   {cm[1][0]:15d}  {cm[1][1]:15d}")
    
    # ROC AUC score
    roc_auc = roc_auc_score(y_test, y_pred_proba)
    print(f"\nROC AUC Score: {roc_auc:.4f}")
    
    # Feature importance (coefficients)
    print("\nFeature Importance (Logistic Regression Coefficients):")
    coefficients = model.coef_[0]
    for name, coef in sorted(zip(feature_names, coefficients), key=lambda x: abs(x[1]), reverse=True):
        direction = "increases" if coef > 0 else "decreases"
        print(f"  {name:30s}: {coef:+.4f} ({direction} fraud risk)")
    
    # Calculate precision/recall at different thresholds
    precision, recall, thresholds = precision_recall_curve(y_test, y_pred_proba)
    
    # Find optimal threshold (maximize F1 score)
    f1_scores = 2 * (precision * recall) / (precision + recall + 1e-10)
    optimal_idx = np.argmax(f1_scores)
    optimal_threshold = thresholds[optimal_idx] if optimal_idx < len(thresholds) else 0.5
    
    print(f"\nOptimal Probability Threshold: {optimal_threshold:.4f}")
    print(f"  (Maximizes F1 score at {f1_scores[optimal_idx]:.4f})")
    
    # Performance at high-risk threshold (75/100 = 0.75 risk score)
    # This is what triggers alerts in production
    high_risk_threshold = 0.75
    y_pred_high_risk = (y_pred_proba >= high_risk_threshold).astype(int)
    
    print(f"\nPerformance at High-Risk Threshold ({high_risk_threshold:.2f}):")
    print(f"  Transactions flagged: {y_pred_high_risk.sum()} ({y_pred_high_risk.mean() * 100:.2f}%)")
    if y_pred_high_risk.sum() > 0:
        precision_hr = np.sum((y_pred_high_risk == 1) & (y_test == 1)) / y_pred_high_risk.sum()
        recall_hr = np.sum((y_pred_high_risk == 1) & (y_test == 1)) / y_test.sum()
        print(f"  Precision: {precision_hr:.4f} (% of alerts that are real fraud)")
        print(f"  Recall: {recall_hr:.4f} (% of fraud cases caught)")
    
    return {
        'roc_auc': roc_auc,
        'optimal_threshold': optimal_threshold,
        'test_accuracy': (y_pred == y_test).mean(),
        'confusion_matrix': cm.tolist()
    }


def save_model(model, feature_names, metrics):
    """Save trained model and metadata."""
    script_dir = os.path.dirname(os.path.abspath(__file__))
    model_dir = os.path.join(script_dir, "..", "model")
    
    # Save model artifact
    model_path = os.path.join(model_dir, f"fraud_model_{MODEL_VERSION}.joblib")
    joblib.dump(model, model_path)
    print(f"\n✓ Model saved to: {model_path}")
    
    # Save metadata
    metadata = {
        'model_version': MODEL_VERSION,
        'model_type': 'LogisticRegression',
        'trained_at': datetime.now().isoformat(),
        'feature_names': feature_names,
        'feature_count': len(feature_names),
        'training_samples': 80000,  # Approximate
        'test_samples': 20000,
        'metrics': metrics,
        'hyperparameters': {
            'class_weight': 'balanced',
            'solver': 'lbfgs',
            'max_iter': 1000,
            'C': 1.0
        },
        'risk_score_mapping': {
            'description': 'fraud_probability (0-1) * 100 = risk_score (0-100)',
            'high_risk_threshold': 75,
            'critical_threshold': 95
        }
    }
    
    metadata_path = os.path.join(model_dir, f"fraud_model_{MODEL_VERSION}_metadata.json")
    with open(metadata_path, 'w') as f:
        json.dump(metadata, f, indent=2)
    
    print(f"✓ Metadata saved to: {metadata_path}")
    
    # Create a symlink to latest model
    latest_model_path = os.path.join(model_dir, "fraud_model_latest.joblib")
    latest_metadata_path = os.path.join(model_dir, "fraud_model_latest_metadata.json")
    
    # Remove existing symlinks if they exist
    for path in [latest_model_path, latest_metadata_path]:
        if os.path.exists(path):
            os.remove(path)
    
    # Create new symlinks
    os.symlink(os.path.basename(model_path), latest_model_path)
    os.symlink(os.path.basename(metadata_path), latest_metadata_path)
    
    print(f"✓ Latest model symlink: {latest_model_path}")


def main():
    """Main training pipeline."""
    print("=" * 60)
    print("Sentinel Fraud Engine - Model Training")
    print(f"Model Version: {MODEL_VERSION}")
    print("=" * 60)
    
    # Load data
    df = load_data()
    
    # Prepare training/test split
    X_train, X_test, y_train, y_test, feature_names = prepare_training_data(df)
    
    # Train model
    model = train_model(X_train, y_train)
    
    # Evaluate model
    metrics = evaluate_model(model, X_test, y_test, feature_names)
    
    # Save model
    save_model(model, feature_names, metrics)
    
    print("\n" + "=" * 60)
    print("Training complete! Model ready for production deployment.")
    print("=" * 60)
    print("\nNext steps:")
    print("1. Review model performance metrics above")
    print("2. Integrate model into risk-engine service")
    print("3. Test model inference with sample transactions")


if __name__ == "__main__":
    main()
