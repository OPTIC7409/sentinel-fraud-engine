#!/usr/bin/env python3
"""
Generate synthetic transaction data for fraud detection model training.

Creates realistic financial transactions with labeled fraud examples.
Features engineered: amount, velocity, location deviation, time anomalies, merchant category risk.
"""

import pandas as pd
import numpy as np
from datetime import datetime, timedelta
import random
import json

# Seed for reproducibility
np.random.seed(42)
random.seed(42)

# Configuration
NUM_TRANSACTIONS = 100000
FRAUD_RATE = 0.05  # 5% fraud rate (realistic for financial systems)
NUM_USERS = 10000
START_DATE = datetime(2026, 1, 1)
END_DATE = datetime(2026, 3, 25)

# Merchant categories with inherent risk levels
MERCHANT_CATEGORIES = {
    'groceries': 0.1,
    'restaurants': 0.2,
    'gas_stations': 0.15,
    'utilities': 0.05,
    'healthcare': 0.08,
    'retail': 0.25,
    'electronics': 0.7,  # High-risk category
    'jewelry': 0.75,     # High-risk category
    'wire_transfer': 0.9,  # Very high-risk
    'crypto': 0.85,      # Very high-risk
    'travel': 0.4,
    'entertainment': 0.3,
}

# Typical transaction amount ranges by category
AMOUNT_RANGES = {
    'groceries': (10, 200),
    'restaurants': (15, 150),
    'gas_stations': (20, 100),
    'utilities': (50, 300),
    'healthcare': (30, 500),
    'retail': (20, 300),
    'electronics': (100, 3000),
    'jewelry': (200, 5000),
    'wire_transfer': (500, 10000),
    'crypto': (100, 10000),
    'travel': (200, 2000),
    'entertainment': (25, 200),
}

# User location patterns (simulating US cities)
USER_LOCATIONS = [
    (40.7128, -74.0060),   # New York
    (34.0522, -118.2437),  # Los Angeles
    (41.8781, -87.6298),   # Chicago
    (29.7604, -95.3698),   # Houston
    (33.4484, -112.0740),  # Phoenix
    (39.7392, -104.9903),  # Denver
    (47.6062, -122.3321),  # Seattle
    (37.7749, -122.4194),  # San Francisco
]


def generate_user_profile(user_id):
    """Generate a user profile with typical behavior patterns."""
    return {
        'user_id': user_id,
        'typical_location': random.choice(USER_LOCATIONS),
        'typical_amount_mean': random.uniform(50, 300),
        'typical_amount_std': random.uniform(20, 100),
        'typical_transaction_rate': random.uniform(2, 15),  # transactions per week
        'preferred_categories': random.sample(list(MERCHANT_CATEGORIES.keys()), k=3)
    }


def haversine_distance(lat1, lon1, lat2, lon2):
    """Calculate distance between two lat/lng points in kilometers."""
    R = 6371  # Earth's radius in km
    
    lat1, lon1, lat2, lon2 = map(np.radians, [lat1, lon1, lat2, lon2])
    dlat = lat2 - lat1
    dlon = lon2 - lon1
    
    a = np.sin(dlat/2)**2 + np.cos(lat1) * np.cos(lat2) * np.sin(dlon/2)**2
    c = 2 * np.arcsin(np.sqrt(a))
    
    return R * c


def is_unusual_time(timestamp):
    """Check if transaction time is unusual (3am-6am or late night)."""
    hour = timestamp.hour
    return hour < 6 or hour > 23


def generate_normal_transaction(user_profile, transaction_id, timestamp):
    """Generate a legitimate transaction following user's typical patterns."""
    category = random.choice(user_profile['preferred_categories'])
    
    # Amount follows user's typical spending pattern
    min_amt, max_amt = AMOUNT_RANGES[category]
    amount = np.clip(
        np.random.normal(user_profile['typical_amount_mean'], user_profile['typical_amount_std']),
        min_amt,
        max_amt
    )
    
    # Location is near user's typical location (within 50km)
    lat_offset = np.random.normal(0, 0.2)  # ~20km variance
    lng_offset = np.random.normal(0, 0.2)
    location = (
        user_profile['typical_location'][0] + lat_offset,
        user_profile['typical_location'][1] + lng_offset
    )
    
    # Timestamp during normal hours (mostly)
    if random.random() < 0.9:  # 90% during normal hours
        if timestamp.hour < 8:
            timestamp = timestamp.replace(hour=random.randint(8, 22))
    
    merchant_id = f"merchant_{category}_{random.randint(1000, 9999)}"
    
    return {
        'transaction_id': transaction_id,
        'user_id': user_profile['user_id'],
        'amount': round(amount, 2),
        'currency': 'USD',
        'merchant_id': merchant_id,
        'merchant_category': category,
        'location_lat': location[0],
        'location_lng': location[1],
        'timestamp': timestamp.isoformat(),
        'is_fraud': 0
    }


def generate_fraudulent_transaction(user_profile, transaction_id, timestamp):
    """Generate a fraudulent transaction with suspicious characteristics."""
    fraud_pattern = random.choice(['high_amount', 'unusual_location', 'rapid_fire', 'high_risk_category'])
    
    if fraud_pattern == 'high_amount':
        # Unusually high transaction amount
        category = random.choice(['electronics', 'jewelry', 'wire_transfer'])
        _, max_amt = AMOUNT_RANGES[category]
        amount = random.uniform(max_amt * 0.8, max_amt * 1.5)
        location = user_profile['typical_location']
        
    elif fraud_pattern == 'unusual_location':
        # Transaction far from typical location
        category = random.choice(list(MERCHANT_CATEGORIES.keys()))
        _, max_amt = AMOUNT_RANGES[category]
        amount = random.uniform(100, max_amt)
        # Pick a random location far away
        location = random.choice([loc for loc in USER_LOCATIONS if loc != user_profile['typical_location']])
        
    elif fraud_pattern == 'rapid_fire':
        # Part of rapid succession of transactions (high velocity)
        category = random.choice(['retail', 'electronics'])
        _, max_amt = AMOUNT_RANGES[category]
        amount = random.uniform(50, max_amt * 0.7)
        location = user_profile['typical_location']
        
    else:  # high_risk_category
        # High-risk merchant category
        category = random.choice(['wire_transfer', 'crypto', 'jewelry'])
        _, max_amt = AMOUNT_RANGES[category]
        amount = random.uniform(max_amt * 0.5, max_amt)
        location = user_profile['typical_location']
    
    # Fraudulent transactions often occur at unusual times
    if random.random() < 0.4:  # 40% at unusual hours
        timestamp = timestamp.replace(hour=random.randint(0, 5))
    
    merchant_id = f"merchant_{category}_{random.randint(1000, 9999)}"
    
    return {
        'transaction_id': transaction_id,
        'user_id': user_profile['user_id'],
        'amount': round(amount, 2),
        'currency': 'USD',
        'merchant_id': merchant_id,
        'merchant_category': category,
        'location_lat': location[0],
        'location_lng': location[1],
        'timestamp': timestamp.isoformat(),
        'is_fraud': 1
    }


def generate_dataset():
    """Generate complete synthetic dataset with fraud labels."""
    print(f"Generating {NUM_TRANSACTIONS} transactions for {NUM_USERS} users...")
    print(f"Target fraud rate: {FRAUD_RATE * 100}%")
    
    # Create user profiles
    user_profiles = {i: generate_user_profile(i) for i in range(1, NUM_USERS + 1)}
    
    transactions = []
    num_fraud = int(NUM_TRANSACTIONS * FRAUD_RATE)
    num_legit = NUM_TRANSACTIONS - num_fraud
    
    # Determine which transactions will be fraudulent
    fraud_indices = set(random.sample(range(NUM_TRANSACTIONS), num_fraud))
    
    # Generate transactions
    for i in range(NUM_TRANSACTIONS):
        # Random timestamp within date range
        delta = END_DATE - START_DATE
        random_seconds = random.randint(0, int(delta.total_seconds()))
        timestamp = START_DATE + timedelta(seconds=random_seconds)
        
        # Random user
        user_id = random.randint(1, NUM_USERS)
        user_profile = user_profiles[user_id]
        
        transaction_id = f"txn_{i+1:08d}"
        
        if i in fraud_indices:
            txn = generate_fraudulent_transaction(user_profile, transaction_id, timestamp)
        else:
            txn = generate_normal_transaction(user_profile, transaction_id, timestamp)
        
        transactions.append(txn)
        
        if (i + 1) % 10000 == 0:
            print(f"Generated {i + 1}/{NUM_TRANSACTIONS} transactions...")
    
    # Convert to DataFrame
    df = pd.DataFrame(transactions)
    
    # Sort by timestamp
    df = df.sort_values('timestamp').reset_index(drop=True)
    
    print(f"\nDataset generated!")
    print(f"Total transactions: {len(df)}")
    print(f"Fraudulent: {df['is_fraud'].sum()} ({df['is_fraud'].mean()*100:.2f}%)")
    print(f"Legitimate: {(df['is_fraud'] == 0).sum()} ({(df['is_fraud'] == 0).mean()*100:.2f}%)")
    
    return df, user_profiles


def engineer_features(df, user_profiles):
    """Engineer features for ML model training."""
    print("\nEngineering features...")
    
    features = []
    
    # Sort by user and timestamp for velocity calculation
    df_sorted = df.sort_values(['user_id', 'timestamp']).copy()
    
    for idx, row in df_sorted.iterrows():
        user_profile = user_profiles[row['user_id']]
        
        # Feature 1: Amount normalized (log scale)
        amount_normalized = np.log(row['amount'] + 1) / np.log(10000)  # Assume max 10k
        
        # Feature 2: Velocity score (transactions in last hour)
        # For training data, we'll simulate this
        # In production, this would query recent transaction history
        velocity_score = np.clip(np.random.poisson(0.5 if row['is_fraud'] == 0 else 2) / 10, 0, 1)
        
        # Feature 3: Location deviation
        distance_km = haversine_distance(
            row['location_lat'], row['location_lng'],
            user_profile['typical_location'][0], user_profile['typical_location'][1]
        )
        location_deviation = np.clip(distance_km / 1000, 0, 1)  # Normalize to 0-1
        
        # Feature 4: Time anomaly
        time_anomaly = 1.0 if is_unusual_time(pd.to_datetime(row['timestamp'])) else 0.0
        
        # Feature 5: Merchant category risk
        merchant_category_risk = MERCHANT_CATEGORIES[row['merchant_category']]
        
        features.append({
            'transaction_id': row['transaction_id'],
            'amount_normalized': amount_normalized,
            'velocity_score': velocity_score,
            'location_deviation': location_deviation,
            'time_anomaly': time_anomaly,
            'merchant_category_risk': merchant_category_risk,
            'is_fraud': row['is_fraud']
        })
    
    features_df = pd.DataFrame(features)
    
    print(f"Features engineered for {len(features_df)} transactions")
    print("\nFeature statistics:")
    print(features_df.describe())
    
    return features_df


def main():
    """Main execution function."""
    print("=" * 60)
    print("Sentinel Fraud Engine - Synthetic Data Generator")
    print("=" * 60)
    
    # Generate raw transactions
    transactions_df, user_profiles = generate_dataset()
    
    # Engineer features
    features_df = engineer_features(transactions_df, user_profiles)
    
    # Save datasets
    import os
    script_dir = os.path.dirname(os.path.abspath(__file__))
    output_dir = os.path.join(script_dir, "data")
    transactions_df.to_csv(f"{output_dir}/transactions.csv", index=False)
    features_df.to_csv(f"{output_dir}/features.csv", index=False)
    
    # Save user profiles as JSON for reference
    with open(f"{output_dir}/user_profiles.json", 'w') as f:
        # Convert tuples to lists for JSON serialization
        profiles_serializable = {
            uid: {**profile, 'typical_location': list(profile['typical_location'])}
            for uid, profile in user_profiles.items()
        }
        json.dump(profiles_serializable, f, indent=2)
    
    print(f"\nData saved to {output_dir}/")
    print(f"  - transactions.csv ({len(transactions_df)} rows)")
    print(f"  - features.csv ({len(features_df)} rows)")
    print(f"  - user_profiles.json ({len(user_profiles)} users)")
    print("\nReady for model training!")


if __name__ == "__main__":
    main()
