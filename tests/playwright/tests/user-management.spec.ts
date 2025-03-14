import { test, expect } from '@playwright/test';
import dotenv from 'dotenv';

dotenv.config();

test.describe('Devtron User Management', () => {
  const apiToken = process.env.API_TOKEN;

  test('should create a new user', async ({ request }) => {
    const newUser = {
      email: 'e2e-test-user@devtron.ai',
      username: 'e2e_test_user',
      role: 'developer',
      password: 'TestPassword123!'
    };

    const response = await request.post('/api/v1/users', {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(newUser)
    });

    expect(response.status()).toBe(201);
    const createdUser = await response.json();
    expect(createdUser.email).toBe(newUser.email);
  });

  test('should retrieve user list', async ({ request }) => {
    const response = await request.get('/api/v1/users', {
      headers: {
        'Authorization': `Bearer ${apiToken}`
      }
    });

    expect(response.status()).toBe(200);
    const users = await response.json();
    expect(Array.isArray(users)).toBeTruthy();
  });

  test('should update user details', async ({ request }) => {
    const userId = 'test-user-id'; // Replace with actual user ID
    const userUpdate = {
      role: 'admin',
      team: 'engineering'
    };

    const response = await request.patch(`/api/v1/users/${userId}`, {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(userUpdate)
    });

    expect(response.status()).toBe(200);
    const updatedUser = await response.json();
    expect(updatedUser.role).toBe(userUpdate.role);
  });

  test('should delete a user', async ({ request }) => {
    const userId = 'test-user-id'; // Replace with actual user ID

    const response = await request.delete(`/api/v1/users/${userId}`, {
      headers: {
        'Authorization': `Bearer ${apiToken}`
      }
    });

    expect(response.status()).toBe(204);
  });

  test('should handle user creation validation errors', async ({ request }) => {
    const invalidUser = {
      email: 'invalid-email', // Invalid email format
      username: '', // Empty username
      role: 'invalid-role'
    };

    const response = await request.post('/api/v1/users', {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(invalidUser)
    });

    expect(response.status()).toBe(400);
    const errorResponse = await response.json();
    expect(errorResponse).toHaveProperty('errors');
  });

  test('should reset user password', async ({ request }) => {
    const passwordResetRequest = {
      email: 'test-user@devtron.ai',
      newPassword: 'NewSecurePassword456!'
    };

    const response = await request.post('/api/v1/users/reset-password', {
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      data: JSON.stringify(passwordResetRequest)
    });

    expect(response.status()).toBe(200);
    const resetResponse = await response.json();
    expect(resetResponse.message).toContain('Password reset successful');
  });
});