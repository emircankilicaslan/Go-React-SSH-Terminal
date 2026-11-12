import React, { useEffect, useRef } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import { WS_BASE, request } from './api';

/**
 * Terminal oturumu. Tek kullanımlık WebSocket bileti, bileşen bağlanırken (efekt içinde) alınır.
 * Böylece React StrictMode geliştirme modunda efekti iki kez çalıştırsa bile her çalıştırma
 * kendi biletini alır ve ilk biletin harcanması bağlantıyı bozmaz.
 */
const TerminalComponent = ({ connectionId, token, onUnauthorized }) => {
    const terminalRef = useRef(null);
    const socketRef = useRef(null);

    useEffect(() => {
        let cancelled = false;

        const term = new Terminal({
            cursorBlink: true,
            fontSize: 14,
            fontFamily: '"Cascadia Code", Menlo, monospace',
            theme: {
                background: '#000000',
                foreground: '#ffffff',
                cursor: '#3b82f6'
            }
        });

        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);
        term.open(terminalRef.current);
        fitAddon.fit();

        term.write('\x1b[33mBağlantı kuruluyor...\x1b[0m\r\n');

        request(`/connections/${connectionId}/ticket`, { token, method: 'POST', onUnauthorized })
            .then(({ ticket }) => {
                if (cancelled) return;
                const socket = new WebSocket(`${WS_BASE}/ws/${ticket}`);
                socket.binaryType = 'arraybuffer';
                socketRef.current = socket;

                socket.onopen = () => {
                    term.write('\r\n\x1b[32m--- WEBSOCKET BAĞLANTISI KURULDU ---\x1b[0m\r\n\r\n');
                };
                socket.onmessage = (event) => {
                    term.write(new Uint8Array(event.data));
                };
                socket.onclose = () => {
                    if (!cancelled) term.write('\r\n\x1b[31m--- BAĞLANTI KESİLDİ ---\x1b[0m\r\n');
                };
                socket.onerror = () => {
                    if (!cancelled) term.write('\r\n\x1b[31m[HATA]: WebSocket bağlantısı başarısız oldu.\x1b[0m\r\n');
                };
            })
            .catch((err) => {
                if (!cancelled) term.write(`\r\n\x1b[31m[HATA]: ${err.message}\x1b[0m\r\n`);
            });

        term.onData((data) => {
            const socket = socketRef.current;
            if (socket && socket.readyState === WebSocket.OPEN) {
                socket.send(data);
            }
        });

        const handleResize = () => fitAddon.fit();
        window.addEventListener('resize', handleResize);

        return () => {
            cancelled = true;
            window.removeEventListener('resize', handleResize);
            socketRef.current?.close();
            socketRef.current = null;
            term.dispose();
        };
    }, [connectionId, token, onUnauthorized]);

    return (
        <div style={{ padding: '15px', background: '#000', borderRadius: '12px', border: '1px solid #334155' }}>
            <div ref={terminalRef} style={{ width: '100%', height: '450px' }} />
        </div>
    );
};

export default TerminalComponent;