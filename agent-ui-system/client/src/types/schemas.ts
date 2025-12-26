export interface UIRequest {
  id: string;
  type: 'confirm' | 'select' | 'form' | 'upload' | 'table' | 'image';
  sessionId: string;
  input: any;
  output?: any;
  status: 'pending' | 'completed' | 'timeout' | 'error';
  createdAt: string;
  completedAt?: string;
  expiresAt: string;
  error?: string;
}

export interface ConfirmInput {
  title: string;
  message?: string;
  approveText?: string;
  rejectText?: string;
}

export interface ConfirmOutput {
  approved: boolean;
  timestamp: string;
  comment?: string;
}

export interface SelectInput {
  title: string;
  options: string[];
  multi?: boolean;
  searchable?: boolean;
}

export interface SelectOutput {
  selected: string | string[];
  comment?: string;
}

export interface FormInput {
  title: string;
  schema: any; // JSON Schema
}

export interface FormOutput {
  data: any;
  comment?: string;
}

export interface UploadInput {
  title: string;
  accept?: string[];
  multiple?: boolean;
  maxSize?: number;
  callbackUrl?: string;
}

export interface UploadOutput {
  files: Array<{
    name: string;
    size: number;
    path: string;
    mimeType: string;
  }>;
  comment?: string;
}

export interface TableInput {
  title: string;
  data: any[];
  columns?: string[];
  multiSelect?: boolean;
  searchable?: boolean;
}

export interface TableOutput {
  selected: any | any[];
  comment?: string;
}

export interface ImageItem {
  src: string; // URL (including /api/images/{id}) or data URI
  alt?: string;
  label?: string;
  caption?: string;
}

export interface ImageInput {
  title: string;
  message?: string;
  images: ImageItem[];
  mode: 'select' | 'confirm';
  options?: string[]; // for the “images as context + multi-select question” variant
  multi?: boolean;
}

export interface ImageOutput {
  selected: number | number[] | boolean | string | string[];
  timestamp: string;
  comment?: string;
}

export interface Notification {
  id: string;
  message: string;
  type: 'info' | 'success' | 'error';
  timestamp: string;
}
