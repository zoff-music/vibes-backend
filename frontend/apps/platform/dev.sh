#!/bin/bash

# Development script for running frontend with HMR
# This script ensures proper HMR setup and handles common development tasks

echo "🚀 Starting Vibez Platform Development Server with HMR"
echo "📍 Frontend will be available at: http://localhost:3001"
echo "🔄 Hot Module Replacement is enabled - changes will update automatically"
echo ""

# Kill any existing processes on our ports
echo "🧹 Cleaning up existing processes..."
lsof -ti :3001,3002 | xargs kill -9 2>/dev/null || true

# Ensure dependencies are up to date
echo "📦 Checking dependencies..."
bun install --silent

# Start the development server
echo "🎵 Starting development server..."
echo "💡 Tip: Make changes to your React components and watch them update instantly!"
echo ""

exec bun run dev