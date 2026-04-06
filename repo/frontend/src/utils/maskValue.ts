export function maskValue(value: string): string {
  if (value.includes('@')) {
    const [local, domain] = value.split('@');
    return local[0] + '***@' + domain[0] + '***' + domain.slice(domain.lastIndexOf('.'));
  }
  if (value.length > 4) {
    return '***' + value.slice(-4);
  }
  return '***';
}
