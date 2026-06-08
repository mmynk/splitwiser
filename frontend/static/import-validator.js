function validateImportData(data) {
  if (typeof data !== 'object' || data === null || Array.isArray(data)) throw 'Invalid JSON: expected an object';
  if (data.total == null || isNaN(parseFloat(data.total))) throw 'Missing required field: "total"';
  if (!Array.isArray(data.participants) || data.participants.length === 0) {
    throw 'Missing required field: "participants" (non-empty array)';
  }

  const participants = data.participants.map((name, i) => {
    if (typeof name !== 'string' || !name.trim()) throw `Participant at index ${i} must be a non-empty string`;
    return name.trim();
  });

  const items = [];
  if (Array.isArray(data.items)) {
    for (const item of data.items) {
      if (typeof item !== 'object' || item === null) throw 'Each item must be an object';
      if (item.amount == null || isNaN(parseFloat(item.amount))) {
        throw `Item "${item.description || '(unnamed)'}" is missing a valid "amount"`;
      }
      items.push({
        description: (item.description || '').trim(),
        amount: parseFloat(item.amount),
      });
    }
  }

  return {
    title: (data.title || '').trim(),
    total: parseFloat(data.total),
    subtotal: data.subtotal != null ? parseFloat(data.subtotal) : parseFloat(data.total),
    participants,
    items,
  };
}

if (typeof module !== 'undefined') module.exports = { validateImportData };
