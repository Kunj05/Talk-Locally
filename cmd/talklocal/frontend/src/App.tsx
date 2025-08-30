import { AlertTriangle, Send } from 'lucide-react'
import { useEffect, useState } from 'react'

function App() {
  const [rooms, setRooms] = useState<string[]>([])
  const [selectedRoom, setSelectedRoom] = useState<string | null>(null)
  const [messages, setMessages] = useState<string[]>([])
  const [message, setMessage] = useState('')

  // Fetch available rooms every 5 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      fetch('http://localhost:9001/available-rooms')
        .then(res => res.json())
        .then(data => setRooms(data))
    }, 5000)

    console.log(rooms);
    
    return () => clearInterval(interval)
  }, [])

  // Fetch messages for selected room
  useEffect(() => {
    if (!selectedRoom) return

    const interval = setInterval(() => {
      fetch(`http://localhost:9001/messages?room=${selectedRoom}`)
        .then(res => res.json())
        .then(data => setMessages(data))
    }, 2000)

    return () => clearInterval(interval)
  }, [selectedRoom])

  const sendMessage = async () => {
    if (message.trim() === '' || !selectedRoom) return

    await fetch(`http://localhost:9001/send?room=${selectedRoom}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message })
    })

    setMessage('')
  }

  // UI if room is not selected
  if (!selectedRoom) {
    return (
      <div className='min-h-screen flex flex-col items-center justify-center bg-gradient-to-tr from-violet-950 to-violet-500 text-white'>
        <h1 className='text-4xl font-bold mb-6'>DisasterNet Rooms</h1>
        <div className='bg-white/10 p-6 rounded-lg w-full max-w-md'>
          <p className='mb-2 font-semibold'>Available Rooms:</p>
          <ul className='mb-4 space-y-2'>
            {rooms.length > 0 ? rooms.map((room, idx) => (
              <li key={idx}>
                <button
                  className='w-full text-left bg-slate-700 px-4 py-2 rounded hover:bg-slate-600'
                  onClick={() => setSelectedRoom(room)}
                >
                  {room}
                </button>
              </li>
            )) : <p>No rooms found. Create one below.</p>}
          </ul>

          <form
            onSubmit={(e) => {
              e.preventDefault()
              const roomInput = (e.target as HTMLFormElement).elements.namedItem('roomName') as HTMLInputElement | null
              const roomName = roomInput?.value || ''
              if (roomName.trim()) {
                setSelectedRoom(roomName.trim())
              }
            }}
          >
            <input
              name="roomName"
              type="text"
              placeholder="Create or join a room"
              className="w-full p-2 rounded bg-slate-800 mb-2"
            />
            <button
              type="submit"
              className="bg-blue-600 w-full py-2 rounded hover:bg-blue-500"
            >
              Join/Create Room
            </button>
          </form>
        </div>
      </div>
    )
  }

  // Main Chat UI
  return (
    <div className='min-h-screen w-full flex flex-col justify-center items-center bg-gradient-to-tr from-violet-950 to-violet-500'>
      <div className='flex flex-col h-[80vh] sm:w-[80vw] lg:w-[60vw] border-4 border-black rounded-3xl overflow-hidden my-8'>
        <div className='flex h-[10%] bg-red-700'>
          <div className='my-auto ml-4'>
            <AlertTriangle className="text-yellow-400 w-12 h-12" />
          </div>
          <div className='text-white ml-4 my-2'>
            <p className='text-3xl font-bold'>DisasterNet (No Internet Required)</p>
            <p className='text-md'>Chatting in Room: <span className="underline">{selectedRoom}</span></p>
          </div>
        </div>

        {/* MESSAGES SECTION */}
        <div className='h-[80%] bg-slate-700 px-6 py-4 overflow-y-auto space-y-2'>
          {messages.length > 0 ? (
            messages.map((msg, idx) => (
              <div key={idx} className='text-white bg-slate-800 px-4 py-2 rounded-md'>
                {msg}
              </div>
            ))
          ) : (
            <p className='text-gray-400'>No messages yet</p>
          )}
        </div>

        {/* INPUT SECTION */}
        <div className='h-[10%] bg-slate-800 flex items-center px-6'>
          <input
            type='text'
            placeholder='Type your message here...'
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            className='w-full p-2 rounded-lg bg-slate-600 text-white'
          />
          <button onClick={sendMessage} className='ml-4'>
            <Send className='text-gray-300 w-8 h-8' />
          </button>
        </div>
      </div>
    </div>
  )
}

export default App
