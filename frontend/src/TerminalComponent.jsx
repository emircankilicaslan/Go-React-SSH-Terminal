import React, { useEffect, useRef } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import { WS_BASE } from './api';

const TerminalComponent = ({ ticket }) => {
    const terminalRef = useRef(null);
    const socketRef = useRef(null);
    const xtermRef = useRef(null);

    useEffect(() => {
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
        xtermRef.current = term;

        term.write('\x1b[33mBağlantı kuruluyor...\x1b[0m\r\n');

        const socket = new WebSocket(`${WS_BASE}/ws/${ticket}`);
        socket.binaryType = "arraybuffer";
        socketRef.current = socket;

        socket.onopen = () => {
            term.write('\r\n\x1b[32m--- WEBSOCKET BAĞLANTISI KURULDU ---\x1b[0m\r\n\r\n');
        };

        socket.onmessage = (event) => {
            const data = new Uint8Array(event.data);
            term.write(data);
        };

        socket.onclose = () => {
            term.write('\r\n\x1b[31m--- BAĞLANTI KESİLDİ ---\x1b[0m\r\n');
        };

        socket.onerror = () => {
            term.write('\r\n\x1b[31m[HATA]: WebSocket bağlantısı başarısız oldu.\x1b[0m\r\n');
        };

        term.onData((data) => {
            if (socketRef.current && socketRef.current.readyState === WebSocket.OPEN) {
                socketRef.current.send(data);
            }
        });

        const handleResize = () => fitAddon.fit();
        window.addEventListener('resize', handleResize);

        return () => {
            window.removeEventListener('resize', handleResize);
            socket.close();
            term.dispose();
        };
    }, [ticket]);

    return (
        <div style={{ padding: '15px', background: '#000', borderRadius: '12px', border: '1px solid #334155' }}>
            <div ref={terminalRef} style={{ width: '100%', height: '450px' }} />
        </div>
    );
};

export default TerminalComponent;