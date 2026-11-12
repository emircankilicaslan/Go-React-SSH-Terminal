import { useState, useEffect, useCallback } from 'react'
import TerminalComponent from './TerminalComponent'
import { request } from './api'
import './App.css'

function App() {
  const [user, setUser] = useState(null)
  const [isRegister, setIsRegister] = useState(false)
  const [authData, setAuthData] = useState({ email: '', password: '' })
  const [connections, setConnections] = useState([])
  const [activeSession, setActiveSession] = useState(null)
  const [serverForm, setServerForm] = useState({ name: '', host: '', port: '22', username: '', password: '' })

  const logout = useCallback(() => {
    setUser(null)
    setConnections([])
    setActiveSession(null)
  }, [])

  const handleAuth = async (e) => {
    e.preventDefault()
    try {
      if (isRegister) {
        await request('/register', { method: 'POST', body: authData })
        alert('Kayıt başarılı! Şimdi giriş yapın.')
        setIsRegister(false)
      } else {
        setUser(await request('/login', { method: 'POST', body: authData }))
      }
    } catch (err) { alert(err.message === 'Failed to fetch' ? 'Bağlantı hatası: backend servisinin çalıştığından emin olun.' : err.message) }
  }

  // Liste, kullanıcı değiştiğinde veya refresh() çağrıldığında yeniden yüklenir.
  const [reloadKey, setReloadKey] = useState(0)
  const fetchConnections = () => setReloadKey(n => n + 1)

  useEffect(() => {
    if (!user) return
    let cancelled = false
    request('/connections', { token: user.token, onUnauthorized: logout })
      .then(list => { if (!cancelled) setConnections(list) })
      .catch(err => console.error('Veri çekme hatası:', err))
    return () => { cancelled = true }
  }, [user, logout, reloadKey])

  const addServer = async (e) => {
    e.preventDefault()
    try {
      await request('/connections', { token: user.token, method: 'POST', body: serverForm, onUnauthorized: logout })
      setServerForm({ name: '', host: '', port: '22', username: '', password: '' })
      fetchConnections()
    } catch (err) { alert(err.message) }
  }

  const deleteServer = async (id) => {
    try {
      await request(`/connections/${id}`, { token: user.token, method: 'DELETE', onUnauthorized: logout })
      fetchConnections()
    } catch (err) { alert(err.message) }
  }

  // Terminal oturumu: tek kullanımlık WebSocket bileti TerminalComponent içinde, bağlanırken alınır.
  const connect = (c) => setActiveSession({ id: c.id, name: c.name })

  if (activeSession) {
    return (
      <div className="terminal-overlay">
        <div className="terminal-header">
          <div className="status-indicator">
            <span className="dot"></span>
            <span>SSH Oturumu: {activeSession.name}</span>
          </div>
          <button className="close-btn" onClick={() => setActiveSession(null)}>Oturumu Kapat ×</button>
        </div>
        <div className="terminal-container">
          <TerminalComponent connectionId={activeSession.id} token={user.token} onUnauthorized={logout} />
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
          <button onClick={logout} className="logout-btn">Çıkış</button>
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
                    <button onClick={() => connect(c)} className="term-btn">Bağlan</button>
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