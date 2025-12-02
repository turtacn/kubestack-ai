
function startDiagnosis() {
    const middleware = document.getElementById('middleware').value;
    const target = document.getElementById('target').value;
    const instance = document.getElementById('instance').value;
    const btn = document.getElementById('start-btn');
    const logsDiv = document.getElementById('logs');

    btn.disabled = true;
    logsDiv.innerHTML = '<div class="log-entry">Starting diagnosis...</div>';

    fetch('/api/v1/diagnose', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            middleware: middleware,
            target: target,
            instance: instance
        })
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        return response.json();
    })
    .then(data => {
        console.log('Diagnosis started:', data);
        logsDiv.innerHTML += `<div class="log-entry">Task ID: ${data.task_id}</div>`;
        connectWS(data.task_id);
    })
    .catch(error => {
        console.error('Error:', error);
        logsDiv.innerHTML += `<div class="log-entry" style="color:red">Error starting diagnosis: ${error}</div>`;
        btn.disabled = false;
    });
}

function connectWS(taskId) {
    const logsDiv = document.getElementById('logs');
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws/diagnose?id=${taskId}`;

    console.log('Connecting to WebSocket:', wsUrl);
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log('WebSocket Connected');
        logsDiv.innerHTML += '<div class="log-entry" style="color:#6a9955">WebSocket Connected. Listening for updates...</div>';
    };

    ws.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);
            console.log('Received:', msg);

            // Check message structure
            // Message format from Handler broadcast: { "topic": "...", "payload": { ... } }
            // Wait, Handler sends the whole Message struct as JSON.
            // Payload can be DiagnosisProgress or generic struct.

            const payload = msg.payload;

            if (payload.Step) {
                // It's a DiagnosisProgress
                const timestamp = new Date().toLocaleTimeString();
                const html = `
                    <div class="log-entry">
                        <span style="color:#888">[${timestamp}]</span>
                        <span class="log-step">[${payload.Step}]</span>
                        <span class="log-status ${payload.Status}">${payload.Status}</span>:
                        <span class="log-msg">${payload.Message}</span>
                    </div>`;
                logsDiv.innerHTML += html;

                if (payload.Step === 'Finished') {
                     document.getElementById('start-btn').disabled = false;
                     // If there's a result link, show it?
                }
            } else if (payload.Type === 'Result') {
                 // Handle result payload if needed
                 logsDiv.innerHTML += `<div class="log-entry" style="color:#6a9955"><strong>Diagnosis Result Received!</strong> (See console for full JSON)</div>`;
                 console.log("Full Result:", payload.Data);
            }

            logsDiv.scrollTop = logsDiv.scrollHeight;

        } catch (e) {
            console.error('Error parsing message:', e);
        }
    };

    ws.onclose = () => {
        console.log('WebSocket Closed');
        logsDiv.innerHTML += '<div class="log-entry" style="color:orange">WebSocket Disconnected</div>';
        // Only re-enable button if we didn't finish gracefully (optional logic)
        // But for now, user might want to run again.
        if (document.getElementById('start-btn').disabled) {
             // Maybe it closed prematurely?
        }
    };

    ws.onerror = (error) => {
        console.error('WebSocket Error:', error);
        logsDiv.innerHTML += '<div class="log-entry" style="color:red">WebSocket Error</div>';
        document.getElementById('start-btn').disabled = false;
    };
}
