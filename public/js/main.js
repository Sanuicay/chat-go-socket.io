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
            const urlParams = new URLSearchParams(window.location.search);
            const chatId = urlParams.get('chatid');
            if (chatId) {
                socket.emit("JoinChatRoom", chatId);  // Send event to server to join the room
                console.log("Joined room:", chatId);
            }
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

        const messageInput = document.getElementById("chat-message");
        const chatContent = document.getElementById("chat-content");
    
        socket.on("NewMessage", (sender, message, createdAt) => {
            console.log("New message received: ", sender, message, createdAt);
            addMessageToChat(sender, message, createdAt, false);
        });

        const urlParams = new URLSearchParams(window.location.search);
        const chatid = urlParams.get('chatid');
        const chatname = urlParams.get('chatname');
        chatContent.innerHTML = "";

        if (chatid && chatname) {
            console.log("chatid: ", chatid);
            socket.emit('ShowMessages', chatid);
            socket.on('ShowMessagesReply', (message, sender, createdAt) => {
                const ChatMessage = [];
                ChatMessage.push({ message, sender, createdAt });

                ChatMessage.forEach((msg) => {
                    const div = document.createElement('div');
                    if (msg.sender === name) {
                        div.innerHTML = `<p><b>You</b>: ${msg.message}</p><p>${msg.createdAt}</p>`;
                    }
                    else {
                        div.innerHTML = `<p><b>${msg.sender}</b>: ${msg.message}</p><p>${msg.createdAt}</p>`;
                    }
                    chatContent.appendChild(div);
                });
            });

            function addMessageToChat(sender, message, createdAt, isPending, tempMessageId = null) {
                // Check for duplicates
                const existingMessage = Array.from(chatContent.querySelectorAll('.message, .pending-message'))
                    .find(el => el.querySelector('p:first-child').textContent.includes(message));

                if (existingMessage) {
                    if (isPending) {
                        updateMessageStatus(existingMessage.id, message, createdAt);
                    }
                    return; 
                }

                const div = document.createElement("div");
                div.classList.add(isPending ? "pending-message" : "message");
                if (tempMessageId) {
                    div.id = tempMessageId;
                }

                const messageContent = `
                    <p><b>${sender === name ? "You" : sender}</b>: ${message}</p>
                    ${isPending ? "" : `<p class="timestamp">${createdAt}</p>`}
                `;

                div.innerHTML = messageContent;
                chatContent.appendChild(div);
                chatContent.scrollTop = chatContent.scrollHeight; 
            }
            
            
            function updateMessageStatus(tempMessageId, message, newContent, isError = false) {
                const messageElement = document.getElementById(tempMessageId);
                if (messageElement) {
                    messageElement.classList.remove("pending-message");
            
                    // Set message class based on error status
                    messageElement.classList.add(isError ? "error-message" : "message"); 
            
                    // Update the message content and add timestamp
                    messageElement.querySelector("p:first-child").textContent = `${messageElement.querySelector("p:first-child").textContent.split(':')[0]}: ${message}`;
                    if (!isError) {
                        messageElement.innerHTML += `<p class="timestamp">${newContent}</p>`;
                    }
                }
            }

            // Send message
            window.sendMessage = () => {
                const message = messageInput.value.trim();
                if (message !== "") {
                    // Optimistically display the message
                    const tempMessageId = 'temp_' + Date.now(); // Temporary ID for the optimistic message
                    addMessageToChat(name, message, "", true, tempMessageId);
                    messageInput.value = "";
        
                    socket.emit("SendMessage", name, chatid, message);
        
                    socket.once("SendMessageReply", (reply) => {
                        if (reply === "Message sent") {
                            // Replace temporary message with the real message once confirmed
                            updateMessageStatus(tempMessageId, message, new Date().toLocaleString(), false); 
                        } else {
                            // Handle error by updating the UI (e.g., display an error message)
                            updateMessageStatus(tempMessageId, message, "Failed to send", true);
                        }
                    });
                } else {
                    alert("Please enter a message");
                }
            };
        } else {
            document.getElementById('chat-content').innerHTML = "Please select a chat to view messages."
        }
    }
}

if (window.location.pathname === '/chat.html') {
    initializeChat();
}

