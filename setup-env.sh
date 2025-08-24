#!/bin/bash

# DayBoard Environment Setup Script
echo "🚀 Setting up DayBoard environment..."

# Create .env file with your credentials
cat > backend/.env << EOF
# DayBoard Production Environment Variables
PORT=8080

# Database - Supabase PostgreSQL
DATABASE_URL=postgresql://postgres:YOUR_PASSWORD@db.YOUR_PROJECT.supabase.co:5432/postgres

# Demo Mode - Set to false for production
DEMO_MODE=false

# Google APIs - Replace with your actual credentials
GOOGLE_CLIENT_ID=your_google_client_id_here
GOOGLE_CLIENT_SECRET=your_google_client_secret_here
GOOGLE_REDIRECT_URI=http://localhost:8080/auth/google/callback
MAPS_API_KEY=your_google_maps_api_key_here

# Plaid API - Replace with your actual credentials
PLAID_CLIENT_ID=your_plaid_client_id_here
PLAID_SECRET=your_plaid_secret_here
PLAID_ENV=sandbox
PLAID_REDIRECT_URI=http://localhost:8080/auth/plaid/callback

# Gemini AI - Replace with your actual API key
GEMINI_API_KEY=your_gemini_api_key_here

# JWT Authentication
JWT_SECRET=dayboard_super_secret_jwt_key_change_in_production_2024
JWT_EXPIRY_HOURS=168

# Application Settings
APP_ENV=development
APP_URL=http://localhost:8080
EOF

echo "✅ Created backend/.env file"
echo ""
echo "📝 Next steps:"
echo "1. Get Google API credentials: https://console.cloud.google.com"
echo "2. Get Plaid API credentials: https://dashboard.plaid.com"
echo "3. Get Gemini API key: https://ai.google.dev"
echo "4. Replace the placeholder values in backend/.env"
echo "5. Run: cd backend && go mod tidy"
echo "6. Run: go run cmd/server/main.go"
echo ""
echo "🔧 For testing with demo mode:"
echo "   Set DEMO_MODE=true in backend/.env"
echo ""
echo "📊 Your database is already configured for Supabase!"
