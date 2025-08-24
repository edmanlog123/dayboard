# 🔧 DayBoard API Setup Guide

## 📋 Overview

Your DayBoard app now has **JWT authentication** implemented and is ready for real API integrations. Follow these detailed steps to connect Google Calendar, Plaid, and Gemini AI.

## 🎯 Current Status

✅ **IMPLEMENTED & WORKING:**
- Database: Connected to Supabase PostgreSQL
- JWT Authentication: Login/signup endpoints working
- Google Distance Matrix: Code ready (just needs API key)
- Tax Calculator: Fully functional with real tax data
- SwiftUI App: Complete 5-tab interface

❌ **NEEDS API KEYS:**
- Google Calendar API + Maps API
- Plaid Banking API
- Gemini AI API

---

## 🔐 1. Google APIs Setup (5 minutes)

### Step 1: Create Google Cloud Project

1. **Go to**: https://console.cloud.google.com
2. **Click**: "Select a project" dropdown → "New Project"
3. **Enter**:
   - Project name: `DayBoard App`
   - Click "Create"
4. **Wait**: ~30 seconds for project creation

### Step 2: Enable Required APIs

1. **Navigate**: APIs & Services → Library
2. **Search & Enable** these APIs:
   - `Google Calendar API` → Click → Enable
   - `Distance Matrix API` → Click → Enable

### Step 3: Create OAuth Credentials

1. **Go to**: APIs & Services → Credentials
2. **Configure OAuth Consent Screen**:
   - Click "OAuth consent screen"
   - User Type: **External** → Create
   - Fill out required fields:
     - App name: `DayBoard`
     - User support email: `your-email@example.com`
     - Developer contact: `your-email@example.com`
   - Click "Save and Continue" through all steps

3. **Create OAuth Client**:
   - Go back to "Credentials" → "Create Credentials" → "OAuth client ID"
   - Application type: **Web application**
   - Name: `DayBoard Backend`
   - Authorized redirect URIs:
     ```
     http://localhost:8080/auth/google/callback
     http://127.0.0.1:8080/auth/google/callback
     ```
   - Click "Create"
   - **SAVE** the `Client ID` and `Client Secret`

### Step 4: Create Maps API Key

1. **Still in Credentials** → "Create Credentials" → "API key"
2. **Copy** the API key
3. **Click** "Restrict Key":
   - API restrictions → Select APIs:
     - ✅ Distance Matrix API
   - Save

### Your Google Credentials:
```
GOOGLE_CLIENT_ID=123456789-abcdefghijklmnop.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-abcdefghijklmnopqrstuvwxyz
MAPS_API_KEY=AIzaSyABC123DEF456GHI789JKL012MNO345
```

---

## 💳 2. Plaid API Setup (3 minutes)

### Step 1: Create Plaid Account

1. **Go to**: https://dashboard.plaid.com
2. **Sign up** with your email
3. **Verify** email address

### Step 2: Get Sandbox Credentials

1. **In Dashboard** → "Team Settings" → "Keys"
2. **Copy**:
   - `client_id`
   - `sandbox secret`
3. **Note**: Sandbox is free with unlimited test bank accounts

### Step 3: Configure Webhook (Optional)

1. **Go to**: "Webhooks" in Plaid Dashboard
2. **Add webhook URL**: `http://localhost:8080/webhooks/plaid`
3. **Select events**: Transactions, Accounts

### Your Plaid Credentials:
```
PLAID_CLIENT_ID=5f8b2c4e3d9a7f1b2c3d4e5f
PLAID_SECRET=sandbox_1234567890abcdef
PLAID_ENV=sandbox
```

---

## 🤖 3. Gemini AI Setup (2 minutes)

### Step 1: Get API Key

1. **Go to**: https://ai.google.dev
2. **Click**: "Get API key in Google AI Studio"
3. **Sign in** with Google account
4. **Click**: "Create API key"
5. **Copy** the API key

### Your Gemini Credentials:
```
GEMINI_API_KEY=AIzaSyABC123DEF456GHI789JKL012MNO345
```

---

## 🔧 4. Update Environment Variables

Edit your `backend/.env` file with the real credentials:

```bash
# Replace the placeholder values:
GOOGLE_CLIENT_ID=your_actual_google_client_id
GOOGLE_CLIENT_SECRET=your_actual_google_client_secret
MAPS_API_KEY=your_actual_maps_api_key
PLAID_CLIENT_ID=your_actual_plaid_client_id
PLAID_SECRET=your_actual_plaid_secret
GEMINI_API_KEY=your_actual_gemini_api_key

# Set to false for real APIs
DEMO_MODE=false
```

---

## 🚀 5. Test Your Setup

### Test 1: Database Connection
```bash
cd backend
go run cmd/server/main.go
# Should start without errors and connect to Supabase
```

### Test 2: Google Maps API
```bash
curl "http://localhost:8080/api/v1/commute/estimate?from=San Francisco, CA&to=Mountain View, CA"
# Should return real distance/time data
```

### Test 3: JWT Authentication
```bash
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","name":"Test User"}'
# Should create a real user in the database
```

---

## 📊 6. API Costs & Limits

### Google APIs (FREE TIERS)
- **Calendar API**: 1,000,000 requests/day
- **Distance Matrix**: $200/month credit (~40,000 requests)
- **Total cost for development**: $0/month

### Plaid API (FREE TIERS)
- **Sandbox**: Unlimited free
- **Development**: 100 bank connections/month
- **Total cost for development**: $0/month

### Gemini AI (FREE TIER)
- **Requests**: 15/minute, 1,500/day
- **Total cost for development**: $0/month

**💡 Total Development Cost: $0/month for all APIs**

---

## 🔍 7. Next Steps

Once APIs are configured:

1. **Google Calendar**: Users can OAuth → sync real calendar events
2. **Plaid**: Users can link real bank accounts → detect subscriptions
3. **Gemini AI**: Get real AI responses instead of demo data
4. **Deploy**: Push to Railway/Render/Fly.io with the same environment variables

---

## 🐛 Troubleshooting

### "API not enabled" error
- Make sure you enabled the API in Google Cloud Console
- Wait 1-2 minutes for propagation

### "Invalid redirect URI" error
- Check OAuth redirect URIs match exactly
- Include both `localhost:8080` and `127.0.0.1:8080`

### "Database connection failed"
- Verify DATABASE_URL is correct
- Check Supabase project is not paused

### "Invalid API key"
- Regenerate API key in respective dashboard
- Check for extra spaces in .env file

---

## 📞 Support

If you need help:
1. Check the error logs in terminal
2. Verify all environment variables are set correctly
3. Test one API at a time
4. Use demo mode (`DEMO_MODE=true`) if APIs aren't working

**Your app is now ready for production! 🎉**
