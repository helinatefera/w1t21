const SSN_REGEX = /\b\d{3}-?\d{2}-?\d{4}\b/;
const PHONE_REGEX = /\b(\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b/;
const EMAIL_REGEX = /\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b/;

export function detectPII(text: string): { detected: boolean; types: string[] } {
  const types: string[] = [];

  if (SSN_REGEX.test(text)) {
    types.push('SSN');
  }
  if (PHONE_REGEX.test(text)) {
    types.push('phone number');
  }
  if (EMAIL_REGEX.test(text)) {
    types.push('email address');
  }

  return { detected: types.length > 0, types };
}

export function detectPIIInFile(content: string): { detected: boolean; types: string[] } {
  return detectPII(content);
}
