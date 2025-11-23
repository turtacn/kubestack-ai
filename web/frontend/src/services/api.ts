import axios from 'axios';

const API_BASE_URL = 'http://localhost:8080/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export interface DiagnosisRequest {
  target: string;
  middleware: string;
  instance: string;
  filters?: Record<string, string>;
}

export interface DiagnosisResponse {
  message: string;
  id: string;
}

export interface DiagnosisResult {
  id: string;
  timestamp: string;
  status: string;
  summary: string;
  issues: Issue[];
}

export interface Issue {
  severity: string;
  description: string;
  root_cause: string;
}

export const DiagnosisAPI = {
  trigger: async (req: DiagnosisRequest): Promise<DiagnosisResponse> => {
    const response = await api.post<DiagnosisResponse>('/diagnosis', req);
    return response.data;
  },
  getResult: async (id: string): Promise<DiagnosisResult> => {
    const response = await api.get<DiagnosisResult>(`/diagnosis/${id}`);
    return response.data;
  },
};

export const AuthAPI = {
  login: async (username: string, password: string): Promise<{ token: string; role: string }> => {
    const response = await api.post('/auth/login', { username, password });
    return response.data;
  },
};
