type PublicKeyResponse = {
  public_key: string;
  algorithm: string;
};

let cachedPublicKey: CryptoKey | null = null;

const pemToArrayBuffer = (pem: string) => {
  const base64 = pem
    .replace(/-----BEGIN PUBLIC KEY-----/g, "")
    .replace(/-----END PUBLIC KEY-----/g, "")
    .replace(/\s/g, "");
  const binary = window.atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes.buffer;
};

const arrayBufferToBase64 = (buffer: ArrayBuffer) => {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  bytes.forEach(byte => {
    binary += String.fromCharCode(byte);
  });
  return window.btoa(binary);
};

const fetchPasswordPublicKey = async () => {
  if (cachedPublicKey) return cachedPublicKey;

  const baseURL = import.meta.env.VITE_API_BASE_URL || "";
  const response = await fetch(`${baseURL}/api/v1/password-public-key`);
  if (!response.ok) throw new Error("Failed to load password public key");
  const data = (await response.json()) as PublicKeyResponse;

  cachedPublicKey = await window.crypto.subtle.importKey(
    "spki",
    pemToArrayBuffer(data.public_key),
    { name: "RSA-OAEP", hash: "SHA-256" },
    false,
    ["encrypt"]
  );

  return cachedPublicKey;
};

export const encryptPassword = async (password: string) => {
  const publicKey = await fetchPasswordPublicKey();
  const encoded = new TextEncoder().encode(password);
  const encrypted = await window.crypto.subtle.encrypt(
    { name: "RSA-OAEP" },
    publicKey,
    encoded
  );
  return arrayBufferToBase64(encrypted);
};
