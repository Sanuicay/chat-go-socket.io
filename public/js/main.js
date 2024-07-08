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

        socket.emit('ShowUsersChat');
        
        socket.on('ShowUsersChatReply', (chatid, chatname) => {
            const ChatList = document.getElementById('ChatList');
            const li = document.createElement('li');
            li.innerHTML = chatname;
            ChatList.appendChild(li);
            li.addEventListener('click', () => {
                window.location.href = 'chat.html?chatid=' + chatid + '&chatname=' + chatname;
            });
        });

        const urlParams = new URLSearchParams(window.location.search);
        const chatid = urlParams.get('chatid');
        const chatname = urlParams.get('chatname');
        const chatContent = document.getElementById("chat-content");
        chatContent.innerHTML = "";

        if (chatid && chatname) {
            console.log("chatid: ", chatid);
            socket.emit('ShowMessages', chatid);
            socket.on('ShowMessagesReply', (message, sender, createdAt) => {
                const ChatMessage = [];
                ChatMessage.push({ message, sender, createdAt });

                ChatMessage.forEach((msg) => {
                    const div = document.createElement('div');
                    // div.innerHTML = `<p><b>${msg.sender}</b>: ${msg.message}</p><p>${msg.createdAt}</p>`;
                    if (msg.sender === name) {
                        div.innerHTML = `<p><b>You</b>: ${msg.message}</p><p>${msg.createdAt}</p>`;
                    }
                    else {
                        div.innerHTML = `<p><b>${msg.sender}</b>: ${msg.message}</p><p>${msg.createdAt}</p>`;
                    }
                    chatContent.appendChild(div);
                });
            });

            // Send message
            window.sendMessage = () => {
                const message = document.getElementById('chat-message').value;
                if (message.trim() !== '') {
                    socket.emit('SendMessage', name, chatid, message);
                    document.getElementById('chat-message').value = '';
                    socket.on('SendMessageReply', (msg) => {
                        if (msg == 'Message sent'){
                            //reload the page
                            window.location.href = 'chat.html?chatid=' + chatid + '&chatname=' + chatname;
                        }
                        else {
                            alert('Message not sent');
                        }
                    });
                } else {
                    alert('Please enter a message');
                }
            }
        } else {
            document.getElementById('chat-content').innerHTML = "Please select a chat to view messages."
        }
    }
}

if (window.location.pathname === '/chat.html') {
    initializeChat();
}
