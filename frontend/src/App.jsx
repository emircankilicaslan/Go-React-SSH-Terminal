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

  const API_BASE = `http://${window.location.hostname}:8080`;

  useEffect(() => {
    if (user) fetchConnections()
  }, [user])

  const handleAuth = async (e) => {
    e.preventDefault()
    const endpoint = isRegister ? '/register' : '/login'
    try {
      const res = await fetch(`${API_BASE}${endpoint}`, {
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
    } catch (err) { alert("Bağlantı Hatası: Backend servisinin çalıştığından emin olun.") }
  }

  const fetchConnections = async () => {
    try {
      const res = await fetch(`${API_BASE}/connections?user_id=${user.user_id}`)
      setConnections(await res.json())
    } catch (err) { console.error("Veri çekme hatası:", err) }
  }

  const addServer = async (e) => {
    e.preventDefault()
    const payload = { ...serverForm, user_id: user.user_id }
    await fetch(`${API_BASE}/connections`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
    setServerForm({ name: '', host: '', port: '22', username: '', password: '' })
    fetchConnections()
  }

  const deleteServer = async (id) => {
    await fetch(`${API_BASE}/connections/${id}`, { method: 'DELETE' })
    fetchConnections()
  }

  if (activeSessionId) {
    return (
      <div className="terminal-overlay">
        <div className="terminal-header">
          <div className="status-indicator">
            <span className="dot"></span>
            <span>SSH Session: {activeSessionId}</span>
          </div>
          <button className="close-btn" onClick={() => setActiveSessionId(null)}>Oturumu Kapat ×</button>
        </div>
        <div className="terminal-container">
          <TerminalComponent sessionId={activeSessionId} />
        </div>
      </div>
    )
  }

  if (!user) {
    return (
      <div className="auth-container">
        <div className="auth-card">
          <div className="brand">
            <span className="logo-icon">🚀</span>
            <h2>{isRegister ? 'Yeni Hesap Oluştur' : 'Sisteme Giriş'}</h2>
          </div>
          
          <form onSubmit={handleAuth} className="auth-form">
            <input type="email" placeholder="E-posta Adresi" value={authData.email} onChange={e => setAuthData({...authData, email: e.target.value})} required />
            <input type="password" placeholder="Şifre" value={authData.password} onChange={e => setAuthData({...authData, password: e.target.value})} required />
            <button type="submit" className="primary-btn">{isRegister ? 'Kayıt Ol' : 'Giriş Yap'}</button>
          </form>
          
          <div className="divider"><span>veya</span></div>

          <button className="github-btn" onClick={() => alert("GitHub entegrasyonu yakında eklenecek.")}>
            <span className="icon">GitHub</span> ile devam et (Yakında)
          </button>

          <p className="toggle-auth" onClick={() => setIsRegister(!isRegister)}>
            {isRegister ? 'Zaten hesabınız var mı? Giriş Yapın' : 'Hesabınız yok mu? Hemen Kayıt Olun'}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="dashboard">
      <header className="dash-header">
        <div className="dash-brand">
          <h1>SSH Terminal <span>Core</span></h1>
        </div>
        <div className="user-nav">
          <span className="user-badge">👤 {user.email}</span>
          <button onClick={() => setUser(null)} className="logout-btn">Çıkış</button>
        </div>
      </header>

      <main className="content">
        <section className="server-form-section">
          <h3>Yeni Sunucu Yapılandırması</h3>
          <form onSubmit={addServer} className="grid-form">
            <input placeholder="Sunucu Adı" value={serverForm.name} onChange={e => setServerForm({...serverForm, name: e.target.value})} />
            <input placeholder="Host IP" value={serverForm.host} onChange={e => setServerForm({...serverForm, host: e.target.value})} />
            <input placeholder="Port" value={serverForm.port} onChange={e => setServerForm({...serverForm, port: e.target.value})} />
            <input placeholder="Kullanıcı" value={serverForm.username} onChange={e => setServerForm({...serverForm, username: e.target.value})} />
            <input type="password" placeholder="SSH Şifresi" value={serverForm.password} onChange={e => setServerForm({...serverForm, password: e.target.value})} />
            <button type="submit" className="add-btn">Sunucuyu Kaydet</button>
          </form>
        </section>

        <section className="server-list-section">
          <h3>Kayıtlı Sunucularınız</h3>
          <div className="server-grid">
            {connections.length === 0 ? (
              <div className="empty-state">Henüz bir sunucu yapılandırılmadı.</div>
            ) : (
              connections.map(c => (
                <div key={c.id} className="server-card">
                  <div className="server-info">
                    <h4>{c.name}</h4>
                    <code>{c.username}@{c.host}:{c.port}</code>
                  </div>
                  <div className="server-actions">
                    <button onClick={() => setActiveSessionId(c.id)} className="term-btn">Bağlan</button>
                    <button onClick={() => deleteServer(c.id)} className="del-btn">Sil</button>
                  </div>
                </div>
              ))
            )}
          </div>
        </section>
      </main>
    </div>
  )
}

export default App