import { test, expect } from '@playwright/test';
import dotenv from 'dotenv';

dotenv.config();

test.describe('Devtron Chart Repository Management', () => {
  const apiToken = process.env.API_TOKEN;

  test('should add a new chart repository', async ({ request }) => {
    const chartRepository = {
      name: 'E2E Test Repo',
      url: 'https://charts.example.com',
      type: 'helm',
      credentials: {
        username: process.env.CHART_REPO_USERNAME || '',
        password: process.env.CHART_REPO_PASSWORD || ''
      }
    };

    const response = await request.post('/api/v1/chart-repositories', {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(chartRepository)
    });

    expect(response.status()).toBe(201);
    const createdRepository = await response.json();
    expect(createdRepository.name).toBe(chartRepository.name);
  });

  test('should list available chart repositories', async ({ request }) => {
    const response = await request.get('/api/v1/chart-repositories', {
      headers: {
        'Authorization': `Bearer ${apiToken}`
      }
    });

    expect(response.status()).toBe(200);
    const repositories = await response.json();
    expect(Array.isArray(repositories)).toBeTruthy();
  });

  test('should update an existing chart repository', async ({ request }) => {
    const repositoryId = 'test-repo-id'; // Replace with actual repository ID
    const updatedRepository = {
      name: 'Updated E2E Test Repo',
      url: 'https://updated-charts.example.com'
    };

    const response = await request.patch(`/api/v1/chart-repositories/${repositoryId}`, {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(updatedRepository)
    });

    expect(response.status()).toBe(200);
    const updatedRepo = await response.json();
    expect(updatedRepo.name).toBe(updatedRepository.name);
  });

  test('should delete a chart repository', async ({ request }) => {
    const repositoryId = 'test-repo-id'; // Replace with actual repository ID

    const response = await request.delete(`/api/v1/chart-repositories/${repositoryId}`, {
      headers: {
        'Authorization': `Bearer ${apiToken}`
      }
    });

    expect(response.status()).toBe(204);
  });

  test('should handle chart repository validation errors', async ({ request }) => {
    const invalidRepository = {
      name: '', // Invalid name
      url: 'invalid-url'
    };

    const response = await request.post('/api/v1/chart-repositories', {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(invalidRepository)
    });

    expect(response.status()).toBe(400);
    const errorResponse = await response.json();
    expect(errorResponse).toHaveProperty('errors');
  });
});