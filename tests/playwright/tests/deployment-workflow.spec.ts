import { test, expect } from '@playwright/test';
import dotenv from 'dotenv';

dotenv.config();

test.describe('Devtron Deployment Workflows', () => {
  const apiToken = process.env.API_TOKEN;

  test('should create a deployment pipeline', async ({ request }) => {
    const deploymentPipeline = {
      name: 'E2E Test Pipeline',
      applicationId: 'test-app-id', // Replace with actual application ID
      environment: 'staging',
      deploymentStrategy: 'rolling-update'
    };

    const response = await request.post('/api/v1/deployment-pipelines', {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(deploymentPipeline)
    });

    expect(response.status()).toBe(201);
    const createdPipeline = await response.json();
    expect(createdPipeline.name).toBe(deploymentPipeline.name);
  });

  test('should trigger a deployment', async ({ request }) => {
    const deploymentTrigger = {
      pipelineId: 'test-pipeline-id', // Replace with actual pipeline ID
      branch: 'main',
      commitHash: 'abc123' // Replace with actual commit hash
    };

    const response = await request.post('/api/v1/deployments', {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(deploymentTrigger)
    });

    expect(response.status()).toBe(202); // Accepted
    const deploymentStatus = await response.json();
    expect(deploymentStatus.status).toBe('in_progress');
  });

  test('should retrieve deployment history', async ({ request }) => {
    const applicationId = 'test-app-id'; // Replace with actual application ID

    const response = await request.get(`/api/v1/applications/${applicationId}/deployments`, {
      headers: {
        'Authorization': `Bearer ${apiToken}`
      }
    });

    expect(response.status()).toBe(200);
    const deploymentHistory = await response.json();
    expect(Array.isArray(deploymentHistory)).toBeTruthy();
  });

  test('should rollback a deployment', async ({ request }) => {
    const rollbackRequest = {
      deploymentId: 'previous-deployment-id', // Replace with actual previous deployment ID
      reason: 'E2E Test Rollback'
    };

    const response = await request.post('/api/v1/deployments/rollback', {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(rollbackRequest)
    });

    expect(response.status()).toBe(200);
    const rollbackStatus = await response.json();
    expect(rollbackStatus.status).toBe('rolled_back');
  });

  test('should handle deployment validation errors', async ({ request }) => {
    const invalidDeploymentTrigger = {
      pipelineId: '', // Invalid pipeline ID
      branch: '',
      commitHash: ''
    };

    const response = await request.post('/api/v1/deployments', {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(invalidDeploymentTrigger)
    });

    expect(response.status()).toBe(400);
    const errorResponse = await response.json();
    expect(errorResponse).toHaveProperty('errors');
  });
});