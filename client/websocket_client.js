function connect() {
    try {
        const ws = new WebSocket('ws://127.0.0.1:56433/ws');
        ws.onmessage = (e) => {
            try {
                window.nursor?.nursor_login(JSON.parse(e.data));
            } catch (e) {}
        };
        ws.onclose = () => setTimeout(connect, 1000);
    } catch (e) {
        setTimeout(connect, 1000);
    }
}

connect(); 