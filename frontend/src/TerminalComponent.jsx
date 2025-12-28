import { useEffect, useRef } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';

const TerminalComponent = ({ sessionId }) => {
    const terminalRef = useRef(null);
    const socketRef = useRef(null);

    useEffect(() => {
        const term = new Terminal({
            cursorBlink: true,
            fontSize: 14,
            theme: { background: '#1e1e1e', foreground: '#ffffff' }
        });

        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);
        term.open(terminalRef.current);
        fitAddon.fit();

        term.write('Connecting...\r\n');

        const socket = new WebSocket(`ws://localhost:8080/ws/${sessionId}`);
        socketRef.current = socket;

        socket.onopen = () => {
            term.write('\r\n\x1b[32m--- SECURE CONNECTION ESTABLISHED ---\x1b[0m\r\n');
        };

        socket.onmessage = (event) => {
            if (typeof event.data === 'string') {
                term.write(event.data);
            } else {
                const reader = new FileReader();
                reader.onload = () => {
                    term.write(reader.result);
                };
                reader.readAsText(event.data);
            }
        };

        socket.onclose = () => {
            term.write('\r\n\x1b[31m--- DISCONNECTED ---\x1b[0m\r\n');
        };

        term.onData((data) => {
            if (socket.readyState === WebSocket.OPEN) {
                socket.send(data);
            }
        });

        return () => {
            socket.close();
            term.dispose();
        };
    }, [sessionId]);

    return (
        <div ref={terminalRef} style={{ width: '100%', height: '100%', textAlign: 'left' }} />
    );
};

export default TerminalComponent;