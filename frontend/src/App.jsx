import { useState, useEffect } from 'react'
import TerminalComponent from './TerminalComponent'
import './App.css'

function App() {
  const [user, setUser] = useState(null)
  const [isRegister, setIsRegister] = useState(false)
  
  const [authData, setAuthData] = useState({ email: '', password: '' })
  
  const [connections, setConnections] = useState([])
  const [activeSessionId, setActiveSessionId] = useState(null)
  const [serverForm, setServerForm] = useState({ name: '', host: '', port: '22', username: '', password: '' })

  useEffect(() => {
    if (user) fetchConnections()
  }, [user])

  const handleAuth = async (e) => {
    e.preventDefault()
    const endpoint = isRegister ? '/register' : '/login'
    try {
      const res = await fetch(`http://localhost:8080${endpoint}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(authData)
      })
      const data = await res.json()
      
      if (!res.ok) {
        alert(data.error)
      } else {
        if (isRegister) {
          alert("Kayıt Başarılı! Şimdi giriş yapın.")
          setIsRegister(false)
        } else {
          setUser(data)
        }
      }
    } catch (err) { alert("Sunucu hatası! Backend çalışıyor mu?") }
  }

  const handleGoogleLogin = () => {
    const confirmLogin = confirm("Google Hesabı ile giriş yapılsın mı? (Simülasyon Modu)")
    
    if (confirmLogin) {
      setTimeout(() => {
        const mockUser = {
          id: 999,
          email: "emir@gmail.com (Google)",
          token: "mock-google-token"
        }
        setUser(mockUser)
        alert("Google ile başarıyla giriş yapıldı! 🚀")
      }, 500)
    }
  }

  const fetchConnections = async () => {
    try {
      const res = await fetch('http://localhost:8080/connections')
      setConnections(await res.json())
    } catch (err) { console.error(err) }
  }

  const addServer = async (e) => {
    e.preventDefault()
    await fetch('http://localhost:8080/connections', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(serverForm)
    })
    setServerForm({ name: '', host: '', port: '22', username: '', password: '' })
    fetchConnections()
  }

  const deleteServer = async (id) => {
    await fetch(`http://localhost:8080/connections/${id}`, { method: 'DELETE' })
    fetchConnections()
  }

  if (activeSessionId) {
    return (
      <div style={{ height: '100vh', background: '#000', display: 'flex', flexDirection: 'column' }}>
        <div style={{ padding: '10px', background: '#333', color: 'white', display: 'flex', justifyContent: 'space-between' }}>
          <span>🟢 Canlı SSH Bağlantısı (ID: {activeSessionId})</span>
          <button onClick={() => setActiveSessionId(null)} style={{ background: 'red', border: 'none', color: 'white', padding: '5px 10px', cursor: 'pointer' }}>Bağlantıyı Kes ❌</button>
        </div>
        <div style={{ flex: 1, padding: '10px' }}><TerminalComponent sessionId={activeSessionId} /></div>
      </div>
    )
  }

  if (!user) {
    return (
      <div style={{ height: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center', background: '#282c34' }}>
        <div style={{ background: 'white', padding: '40px', borderRadius: '10px', width: '350px', boxShadow: '0 4px 10px rgba(0,0,0,0.2)' }}>
          <h2 style={{ textAlign: 'center', color: '#333' }}>{isRegister ? 'Kayıt Ol' : 'Giriş Yap'}</h2>
          
          <form onSubmit={handleAuth} style={{ display: 'flex', flexDirection: 'column', gap: '15px' }}>
            <input type="email" placeholder="Email" value={authData.email} onChange={e => setAuthData({...authData, email: e.target.value})} style={inputStyle} required />
            <input type="password" placeholder="Şifre" value={authData.password} onChange={e => setAuthData({...authData, password: e.target.value})} style={inputStyle} required />
            <button type="submit" style={btnStyle}>{isRegister ? 'Kayıt Ol' : 'Giriş Yap'}</button>
          </form>
          
          <div style={{ marginTop: '15px', pt: '15px', borderTop: '1px solid #eee' }}>
            <button onClick={handleGoogleLogin} style={{ ...btnStyle, background: '#db4437', width: '100%' }}>
              Google ile Giriş Yap
            </button>
          </div>

          <p style={{ textAlign: 'center', marginTop: '15px', color: '#666', fontSize: '14px', cursor: 'pointer' }} onClick={() => setIsRegister(!isRegister)}>
            {isRegister ? 'Hesabın var mı? Giriş Yap' : 'Hesabın yok mu? Kayıt Ol'}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div style={{ maxWidth: '800px', margin: '20px auto', padding: '20px', fontFamily: 'Arial' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <h1 style={{ margin: 0 }}>🚀 Panel</h1>
        </div>
        <div>
          <span style={{ marginRight: '10px', fontWeight: 'bold' }}>👤 {user.email}</span>
          <button onClick={() => setUser(null)} style={{ background: '#666', color: 'white', border: 'none', padding: '8px 15px', borderRadius: '5px', cursor: 'pointer' }}>Çıkış Yap</button>
        </div>
      </div>

      <div style={{ background: '#f8f9fa', padding: '20px', borderRadius: '8px', marginBottom: '20px', border: '1px solid #ddd' }}>
        <h3>Yeni Sunucu Ekle</h3>
        <form onSubmit={addServer} style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
          <input placeholder="Sunucu İsmi (Örn: Web01)" value={serverForm.name} onChange={e => setServerForm({...serverForm, name: e.target.value})} style={inputStyle} />
          <input placeholder="Host IP (Örn: 192.168.1.5)" value={serverForm.host} onChange={e => setServerForm({...serverForm, host: e.target.value})} style={inputStyle} />
          <input placeholder="Port (Genelde 22)" value={serverForm.port} onChange={e => setServerForm({...serverForm, port: e.target.value})} style={inputStyle} />
          <input placeholder="Kullanıcı Adı (root)" value={serverForm.username} onChange={e => setServerForm({...serverForm, username: e.target.value})} style={inputStyle} />
          <input type="password" placeholder="Şifre" value={serverForm.password} onChange={e => setServerForm({...serverForm, password: e.target.value})} style={inputStyle} />
          <button type="submit" style={{ ...btnStyle, gridColumn: 'span 2', background: '#28a745', fontSize: '16px' }}>Sunucuyu Ekle +</button>
        </form>
      </div>

      <h3>Kayıtlı Sunucular</h3>
      {connections.length === 0 ? <p style={{ color: '#888', textAlign: 'center' }}>Henüz eklenmiş sunucu yok.</p> : (
        <div style={{ display: 'grid', gap: '10px' }}>
          {connections.map(c => (
            <div key={c.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', background: 'white', padding: '15px', border: '1px solid #ddd', borderRadius: '5px', boxShadow: '0 2px 5px rgba(0,0,0,0.05)' }}>
              <div>
                <strong style={{ fontSize: '18px' }}>{c.name}</strong>
                <div style={{ color: '#666', fontSize: '14px', marginTop: '4px' }}>{c.username}@{c.host}:{c.port}</div>
              </div>
              <div>
                <button onClick={() => setActiveSessionId(c.id)} style={{ ...btnStyle, background: '#007bff', marginRight: '10px' }}>Terminali Aç 📟</button>
                <button onClick={() => deleteServer(c.id)} style={{ ...btnStyle, background: '#dc3545' }}>Sil 🗑️</button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

const inputStyle = { padding: '10px', border: '1px solid #ddd', borderRadius: '5px', fontSize: '14px' }
const btnStyle = { padding: '10px', background: '#333', color: 'white', border: 'none', borderRadius: '5px', cursor: 'pointer', fontWeight: 'bold' }

export default App