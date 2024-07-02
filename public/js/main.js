const socket = io('http://localhost:3000', (autoConnect = false));
socket.on('reply', (msg) => {
    const chat = document.getElementById('chat');
    const newMessage = document.createElement('div');
    newMessage.textContent = `Server: ${msg}`;
    chat.appendChild(newMessage);
});

socket.on('connect', () => {
        console.log('Connected to server');
    });

socket.on('disconnect', () => {
    console.log('Disconnected from server');
});

socket.on('registerReply', (msg) => {
    alert(msg);
});

socket.on('loginReply', (msg) => {
    alert(msg);
});

socket.on('loginReplySuccess', (msg) => {
    alert(msg);
    window.location.href = '/index.html';
});

socket.on('loginReplyFail', (msg) => {
    alert(msg);
    //refresh page
    location.reload();
});
