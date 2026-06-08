export interface ImportedBillItem {
  description: string;
  amount: number;
}

export interface ImportedBill {
  title: string;
  total: number;
  subtotal: number;
  participants: string[];
  items: ImportedBillItem[];
}

function isPlainObject(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

export function validateImportData(data: unknown): ImportedBill {
  if (!isPlainObject(data)) {
    throw new Error('Invalid JSON: expected an object');
  }

  if (data.total == null || isNaN(parseFloat(String(data.total)))) {
    throw new Error('Missing required field: "total"');
  }

  if (!Array.isArray(data.participants) || data.participants.length === 0) {
    throw new Error('Missing required field: "participants" (non-empty array)');
  }

  const participants = data.participants.map((name, i) => {
    if (typeof name !== 'string' || !name.trim()) {
      throw new Error(`Participant at index ${i} must be a non-empty string`);
    }
    return name.trim();
  });

  const items: ImportedBillItem[] = [];
  if (Array.isArray(data.items)) {
    for (const raw of data.items) {
      if (!isPlainObject(raw)) {
        throw new Error('Each item must be an object');
      }
      if (raw.amount == null || isNaN(parseFloat(String(raw.amount)))) {
        const label = typeof raw.description === 'string' && raw.description ? raw.description : '(unnamed)';
        throw new Error(`Item "${label}" is missing a valid "amount"`);
      }
      items.push({
        description: typeof raw.description === 'string' ? raw.description.trim() : '',
        amount: parseFloat(String(raw.amount)),
      });
    }
  }

  return {
    title: typeof data.title === 'string' ? data.title.trim() : '',
    total: parseFloat(String(data.total)),
    subtotal: data.subtotal != null ? parseFloat(String(data.subtotal)) : parseFloat(String(data.total)),
    participants,
    items,
  };
}
