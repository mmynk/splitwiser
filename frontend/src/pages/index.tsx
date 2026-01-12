import { useState } from 'react'
import { createPromiseClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { SplitService } from '../api/generated/splitwiser_connect'
import type { CalculateSplitResponse } from '../api/generated/splitwiser_pb'

// Create Connect transport
const transport = createConnectTransport({
  baseUrl: 'http://localhost:8080',
})

// Create client
const client = createPromiseClient(SplitService, transport)

export default function Home() {
  const [billTotal, setBillTotal] = useState('')
  const [billSubtotal, setBillSubtotal] = useState('')
  const [participants, setParticipants] = useState([''])
  const [items, setItems] = useState<Array<{ description: string; amount: string; assignedTo: string[] }>>([])
  const [splitResult, setSplitResult] = useState<CalculateSplitResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  const addParticipant = () => {
    setParticipants([...participants, ''])
  }

  const removeParticipant = (index: number) => {
    if (participants.length === 1) return // Keep at least one
    const newParticipants = participants.filter((_, i) => i !== index)
    setParticipants(newParticipants)

    // Remove this participant from all item assignments
    const participantName = participants[index]
    setItems(items.map(item => ({
      ...item,
      assignedTo: item.assignedTo.filter(p => p !== participantName)
    })))
  }

  const updateParticipant = (index: number, value: string) => {
    const oldValue = participants[index]
    const newParticipants = [...participants]
    newParticipants[index] = value
    setParticipants(newParticipants)

    // Update participant name in item assignments
    if (oldValue && value) {
      setItems(items.map(item => ({
        ...item,
        assignedTo: item.assignedTo.map(p => p === oldValue ? value : p)
      })))
    }
  }

  const addItem = () => {
    setItems([...items, { description: '', amount: '', assignedTo: participants.filter(p => p.trim()) }])
  }

  const removeItem = (index: number) => {
    setItems(items.filter((_, i) => i !== index))
  }

  const updateItem = (index: number, field: string, value: any) => {
    const newItems = [...items]
    newItems[index] = { ...newItems[index], [field]: value }
    setItems(newItems)
  }

  const toggleItemAssignment = (itemIndex: number, participant: string) => {
    const newItems = [...items]
    const item = newItems[itemIndex]
    if (item.assignedTo.includes(participant)) {
      item.assignedTo = item.assignedTo.filter(p => p !== participant)
    } else {
      item.assignedTo = [...item.assignedTo, participant]
    }
    setItems(newItems)
  }

  const calculateSplit = async () => {
    setError(null)
    setSplitResult(null)

    // Default empty descriptions to "Item 1", "Item 2", etc.
    const itemsWithDefaults = items.map((item, index) => ({
      description: item.description.trim() || `Item ${index + 1}`,
      amount: parseFloat(item.amount) || 0,
      assignedTo: item.assignedTo.filter(p => p.trim())
    }))

    try {
      const response = await client.calculateSplit({
        items: itemsWithDefaults,
        total: parseFloat(billTotal),
        subtotal: parseFloat(billSubtotal),
        participants: participants.filter(p => p.trim())
      })

      setSplitResult(response)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to calculate split')
      console.error('Error calculating split:', err)
    }
  }

  return (
    <div style={{ maxWidth: '800px', margin: '0 auto', padding: '20px', fontFamily: 'system-ui, sans-serif' }}>
      <h1>Splitwiser</h1>
      <p style={{ color: '#666' }}>Split bills with item-level granularity and automatic tax distribution</p>

      <section style={{ marginTop: '30px' }}>
        <h2>Bill Details</h2>
        <div style={{ display: 'flex', gap: '10px', marginBottom: '10px' }}>
          <input
            type="number"
            placeholder="Total (with tax)"
            value={billTotal}
            onChange={(e) => setBillTotal(e.target.value)}
            style={{ flex: 1, padding: '8px', fontSize: '16px' }}
          />
          <input
            type="number"
            placeholder="Subtotal (before tax)"
            value={billSubtotal}
            onChange={(e) => setBillSubtotal(e.target.value)}
            style={{ flex: 1, padding: '8px', fontSize: '16px' }}
          />
        </div>
        {billTotal && billSubtotal && (
          <div style={{ color: '#666', fontSize: '14px' }}>
            Tax: ${(parseFloat(billTotal) - parseFloat(billSubtotal)).toFixed(2)}
          </div>
        )}
      </section>

      <section style={{ marginTop: '30px' }}>
        <h2>Participants</h2>
        {participants.map((participant, index) => (
          <div key={index} style={{ marginBottom: '10px', display: 'flex', gap: '10px' }}>
            <input
              type="text"
              placeholder={`Person ${index + 1}`}
              value={participant}
              onChange={(e) => updateParticipant(index, e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && addParticipant()}
              style={{ flex: 1, padding: '8px', fontSize: '16px' }}
            />
            {participants.length > 1 && (
              <button
                onClick={() => removeParticipant(index)}
                style={{ padding: '8px 16px', cursor: 'pointer', backgroundColor: '#fee', border: '1px solid #fcc', borderRadius: '4px' }}
              >
                Remove
              </button>
            )}
          </div>
        ))}
        <button onClick={addParticipant} style={{ padding: '8px 16px', cursor: 'pointer' }}>
          + Add Participant
        </button>
      </section>

      <section style={{ marginTop: '30px' }}>
        <h2>Items</h2>
        {items.map((item, itemIndex) => (
          <div key={itemIndex} style={{ marginBottom: '20px', padding: '15px', border: '1px solid #ddd', borderRadius: '4px', position: 'relative' }}>
            <div style={{ display: 'flex', gap: '10px', marginBottom: '10px' }}>
              <input
                type="text"
                placeholder={`Item ${itemIndex + 1} (optional)`}
                value={item.description}
                onChange={(e) => updateItem(itemIndex, 'description', e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && addItem()}
                style={{ flex: 2, padding: '8px', fontSize: '16px' }}
              />
              <input
                type="number"
                placeholder="Amount"
                value={item.amount}
                onChange={(e) => updateItem(itemIndex, 'amount', e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && addItem()}
                style={{ flex: 1, padding: '8px', fontSize: '16px' }}
              />
              <button
                onClick={() => removeItem(itemIndex)}
                style={{ padding: '8px 16px', cursor: 'pointer', backgroundColor: '#fee', border: '1px solid #fcc', borderRadius: '4px' }}
              >
                Remove
              </button>
            </div>
            <div style={{ fontSize: '14px', marginBottom: '5px', color: '#666' }}>Assigned to:</div>
            <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
              {participants.filter(p => p.trim()).map((participant, pIndex) => (
                <label key={pIndex} style={{ display: 'flex', alignItems: 'center', gap: '5px', cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={item.assignedTo.includes(participant)}
                    onChange={() => toggleItemAssignment(itemIndex, participant)}
                  />
                  {participant}
                </label>
              ))}
            </div>
          </div>
        ))}
        <button onClick={addItem} style={{ padding: '8px 16px', cursor: 'pointer' }}>
          + Add Item
        </button>
      </section>

      <section style={{ marginTop: '30px' }}>
        <button
          onClick={calculateSplit}
          style={{
            padding: '12px 24px',
            fontSize: '16px',
            backgroundColor: '#0070f3',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer'
          }}
        >
          Calculate Split
        </button>
      </section>

      {error && (
        <section style={{ marginTop: '30px', padding: '15px', backgroundColor: '#fee', border: '1px solid #fcc', borderRadius: '4px' }}>
          <h3 style={{ margin: '0 0 10px 0', color: '#c00' }}>Error</h3>
          <p style={{ margin: 0 }}>{error}</p>
        </section>
      )}

      {splitResult && (
        <section style={{ marginTop: '30px', padding: '20px', backgroundColor: '#f9f9f9', border: '1px solid #ddd', borderRadius: '4px' }}>
          <h2>Split Results</h2>
          <div style={{ marginBottom: '15px', fontSize: '14px', color: '#666' }}>
            <div>Subtotal: ${splitResult.subtotal.toFixed(2)}</div>
            <div>Tax: ${splitResult.taxAmount.toFixed(2)}</div>
          </div>

          {Object.entries(splitResult.splits).map(([person, split]) => (
            <div key={person} style={{ marginBottom: '15px', padding: '15px', backgroundColor: 'white', border: '1px solid #ddd', borderRadius: '4px' }}>
              <h3 style={{ margin: '0 0 10px 0' }}>{person}</h3>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '10px', fontSize: '14px' }}>
                <div>
                  <div style={{ color: '#666' }}>Subtotal</div>
                  <div style={{ fontWeight: 'bold' }}>${split.subtotal.toFixed(2)}</div>
                </div>
                <div>
                  <div style={{ color: '#666' }}>Tax</div>
                  <div style={{ fontWeight: 'bold' }}>${split.tax.toFixed(2)}</div>
                </div>
                <div>
                  <div style={{ color: '#666' }}>Total</div>
                  <div style={{ fontWeight: 'bold', fontSize: '16px', color: '#0070f3' }}>${split.total.toFixed(2)}</div>
                </div>
              </div>
            </div>
          ))}
        </section>
      )}
    </div>
  )
}
