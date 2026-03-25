#!/usr/bin/env python3
"""stdin argv[1] = JSON features -> stdout JSON {probability}. Invoked from Go risk engine."""

import sys
import json
import joblib
import numpy as np
from pathlib import Path

MODEL_DIR = Path(__file__).parent
MODEL_PATH = MODEL_DIR / "fraud_model_v1.0.0.joblib"

try:
    model = joblib.load(MODEL_PATH)
except Exception as e:
    print(json.dumps({"error": f"Failed to load model: {str(e)}"}), file=sys.stderr)
    sys.exit(1)

def predict(features_dict):
    # Column order must match train_model.py / scorer.go
    feature_order = [
        'amount_normalized',
        'velocity_score',
        'location_deviation',
        'time_anomaly',
        'merchant_category_risk'
    ]
    
    # Extract features in correct order
    feature_vector = np.array([[features_dict[f] for f in feature_order]])
    
    # Predict probability (column 1 is fraud class)
    fraud_prob = model.predict_proba(feature_vector)[0][1]
    
    return float(fraud_prob)

def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "No features provided"}), file=sys.stderr)
        sys.exit(1)
    
    try:
        # Parse features from JSON argument
        features_json = sys.argv[1]
        features = json.loads(features_json)
        
        # Run inference
        probability = predict(features)
        
        # Output result as JSON
        result = {"probability": probability}
        print(json.dumps(result))
        
    except Exception as e:
        print(json.dumps({"error": str(e)}), file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
