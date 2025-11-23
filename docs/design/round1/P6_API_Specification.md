# P6: Web API Specification

## Overview
This document specifies the REST API and WebSocket interfaces for the KubeStack-AI Web UI.

## Authentication
All API endpoints (except `/api/v1/auth/login`) require a JWT token in the `Authorization` header:
`Authorization: Bearer <token>`

## Base URL
`http://localhost:8080/api/v1`

## Endpoints

### 1. Authentication
#### Login
- **POST** `/auth/login`
- **Body**: `{ "username": "admin", "password": "..." }`
- **Response**: `{ "token": "eyJ...", "role": "admin" }`

### 2. Diagnosis
#### Trigger Diagnosis
- **POST** `/diagnosis`
- **Permission**: `diagnosis:write`
- **Body**:
  ```json
  {
    "target": "redis-cluster-prod",
    "middleware": "redis",
    "instance": "redis-main"
  }
  ```
- **Response**: `202 Accepted`
  ```json
  {
    "message": "Diagnosis started",
    "id": "redis-cluster-prod"
  }
  ```

#### Get Diagnosis Result
- **GET** `/diagnosis/:id`
- **Permission**: `diagnosis:read`
- **Response**:
  ```json
  {
    "id": "...",
    "status": "Completed",
    "issues": [...]
  }
  ```

### 3. Execution (Placeholder)
#### Execute Plan
- **POST** `/execution/plan/:id/execute`
- **Permission**: `execution:write`

### 4. Configuration
#### Get Config
- **GET** `/config`
- **Permission**: `diagnosis:read`

#### Update Config
- **PUT** `/config`
- **Permission**: `diagnosis:write`
- **Body**: (Full Config Object)

## WebSocket

### Diagnostic Stream
- **URL**: `ws://localhost:8080/ws/diagnosis/:id`
- **Events**:
  - `topic`: Matching the diagnosis ID (or target)
  - `payload`:
    ```json
    {
      "step": "Data Collection",
      "status": "InProgress",
      "message": "Collecting metrics..."
    }
    ```
