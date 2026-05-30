import axios from "axios";

export const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE ?? "http://localhost:8080",
  timeout: 30000,
});

http.interceptors.response.use(
  (resp) => resp,
  (error) => {
    const message =
      error.response?.data?.error?.message ??
      error.response?.data?.message ??
      error.message ??
      "Request failed";
    return Promise.reject(new Error(message));
  },
);
