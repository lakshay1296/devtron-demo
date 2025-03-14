import { test, expect } from '@playwright/test';
import dotenv from 'dotenv';

dotenv.config();

test.describe('Devtron API Endpoints', () => {
  const apiToken = process.env.API_TOKEN;

  test('should retrieve application list', async ({ request }) => {
    const response = await request.get('/api/v1/applications', {
      headers: {
        'Authorization': `Bearer ${apiToken}`
      }
    });

    expect(response.status()).toBe(200);
    const applications = await response.json();
    expect(Array.isArray(applications)).toBeTruthy();
  });

  test('should create a new application', async ({ request }) => {
    const newApplication = {
      name: 'Test Application',
      description: 'E2E Test Application',
      type: 'microservice'
    };

    const response = await request.post('/api/v1/applications', {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(newApplication)
    });

    expect(response.status()).toBe(201);
    const createdApplication = await response.json();
    expect(createdApplication.name).toBe(newApplication.name);
  });

  test('should retrieve application details', async ({ request }) => {
    const applicationId = 'test-app-id'; // Replace with actual method to get an application ID

    const response = await request.get(`/api/v1/applications/${applicationId}`, {
      headers: {
        'Authorization': `Bearer ${apiToken}`
      }
    });

    expect(response.status()).toBe(200);
    const applicationDetails = await response.json();
    expect(applicationDetails.id).toBe(applicationId);
  });

  test('should update application configuration', async ({ request }) => {
    const applicationId = 'test-app-id'; // Replace with actual method to get an application ID
    const updatedConfig = {
      description: 'Updated E2E Test Application',
      environment: 'staging'
    };

    const response = await request.patch(`/api/v1/applications/${applicationId}`, {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(updatedConfig)
    });

    expect(response.status()).toBe(200);
    const updatedApplication = await response.json();
    expect(updatedApplication.description).toBe(updatedConfig.description);
  });

  test('should handle API authentication errors', async ({ request }) => {
    const response = await request.get('/api/v1/applications', {
      headers: {
        'Authorization': 'Bearer invalid_token'
      }
    });

    expect(response.status()).toBe(401);
  });
});