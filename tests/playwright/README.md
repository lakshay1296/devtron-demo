# Devtron E2E Testing Framework

## Overview
Comprehensive Playwright-based End-to-End Testing Suite for Devtron

## Prerequisites
- Node.js 16+
- Playwright
- Docker
- Kubernetes Cluster

## Setup
1. Install dependencies
```bash
npm init -y
npm install @playwright/test
npx playwright install
```

## Test Categories
1. Authentication Tests
2. API Endpoint Tests
3. User Management Tests
4. Deployment Workflow Tests
5. Chart Repository Tests
6. Security Configuration Tests

## Running Tests
```bash
npx playwright test
```

## Test Configuration
- Parallel execution
- Detailed reporting
- Cross-browser testing