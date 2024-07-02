// main.js

function startChat() {
    const name = document.getElementById('name').value;
    if (name.trim() !== '') {
        const sessionID = 'session_' + Date.now() + '_' + Math.floor(Math.random() * (10 ** 10));
        localStorage.setItem('sessionID', sessionID);
        localStorage.setItem('name', name);
        window.location.href = 'chat.html';
    } else {
        alert('Please enter your name');
    }
}

function initializeChat() {
    const sessionID = localStorage.getItem('sessionID');
    const name = localStorage.getItem('name');

    if (!sessionID || !name) {
        alert('Session not found. Please start a new chat.');
        window.location.href = 'index.html';
    } else {
        const socket = io('http://localhost:3000', {
            query: {
                sessionID: sessionID,
                name: name
            }
        });

        socket.on('connect', () => {
            console.log('Connected to server');
        });

        socket.on('disconnect', () => {
            console.log('Disconnected from server');
        });

        socket.on('message', (data) => {
            const chatBox = document.getElementById('chat-box');
            chatBox.innerHTML += `<p><strong>${data.name}:</strong> ${data.message}</p>`;
        });

        window.sendMessage = function() {
            const message = document.getElementById('message').value;
            if (message.trim() !== '') {
                socket.emit('message', { name: name, message: message });
                document.getElementById('message').value = '';
            } else {
                alert('Please enter a message');
            }
        }
    }
}

if (window.location.pathname === '/chat.html') {
    initializeChat();
}
