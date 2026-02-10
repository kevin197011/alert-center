import dayjs from 'dayjs';

export interface ExportColumn {
  title: string;
  dataIndex: string;
  render?: (value: unknown, record: unknown) => string | number;
}

export function exportToExcel<T>(
  data: T[],
  columns: ExportColumn[],
  filename: string
): void {
  const headers = columns.map(col => col.title);
  const keys = columns.map(col => col.dataIndex);

  const rows = data.map(item => {
    return keys.map(key => {
      const value = (item as Record<string, unknown>)[key];
      const column = columns.find(col => col.dataIndex === key);
      if (column?.render && value !== undefined && value !== null) {
        return column.render(value, item);
      }
      return value as string | number | undefined;
    });
  });

  const csvContent = [
    headers.join(','),
    ...rows.map(row =>
      row.map(cell => {
        const str = String(cell ?? '');
        if (str.includes(',') || str.includes('"') || str.includes('\n')) {
          return `"${str.replace(/"/g, '""')}"`;
        }
        return str;
      }).join(',')
    ),
  ].join('\n');

  downloadFile(csvContent, `${filename}_${dayjs().format('YYYYMMDDHHmmss')}.csv`, 'text/csv;charset=utf-8;');
}

export function exportToCSV<T>(
  data: T[],
  columns: ExportColumn[],
  filename: string
): void {
  exportToExcel(data, columns, filename);
}

export function exportToJSON<T>(
  data: T[],
  filename: string
): void {
  const jsonContent = JSON.stringify(data, null, 2);
  downloadFile(jsonContent, `${filename}_${dayjs().format('YYYYMMDDHHmmss')}.json`, 'application/json');
}

function downloadFile(content: string, filename: string, mimeType: string): void {
  const blob = new Blob([new Uint8Array([0xEF, 0xBB, 0xBF]), content], { type: mimeType });
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
}

export function formatDuration(seconds: number): string {
  if (seconds < 60) {
    return `${seconds.toFixed(0)}秒`;
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return `${minutes}分钟`;
  }
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  if (hours < 24) {
    return `${hours}小时${remainingMinutes}分钟`;
  }
  const days = Math.floor(hours / 24);
  const remainingHours = hours % 24;
  return `${days}天${remainingHours}小时`;
}

export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function downloadBlob(blob: Blob, filename: string): void {
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
}
