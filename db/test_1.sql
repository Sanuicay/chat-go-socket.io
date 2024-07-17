    -- Demo 1: Private chat and Group chat
    -- Create database messaging_app
    CREATE DATABASE messaging_app;
    USE messaging_app;

    -- Create table for users
    CREATE TABLE users (
        id INT PRIMARY KEY AUTO_INCREMENT,
        name VARCHAR(100) NOT NULL,
        is_online BOOLEAN DEFAULT FALSE
    );

    -- Create table for messages
    CREATE TABLE messages (
        id INT PRIMARY KEY AUTO_INCREMENT,
        sender_id INT NOT NULL,
        chat_id INT,
        message TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Create table for message_status
    CREATE TABLE message_status (
        id INT PRIMARY KEY AUTO_INCREMENT,
        message_id INT NOT NULL,
        receiver_id INT NOT NULL,
        status ENUM('sent', 'delivered', 'read', 'error') DEFAULT 'sent',
        status_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Create table for chat
    CREATE TABLE chat (
        id INT PRIMARY KEY AUTO_INCREMENT,
        name VARCHAR(100) NOT NULL,
        is_group BOOLEAN DEFAULT FALSE
    );

    -- Create table for chat_users
    CREATE TABLE chat_users (
        id INT PRIMARY KEY AUTO_INCREMENT,
        chat_id INT NOT NULL,
        user_id INT NOT NULL
    );

    -- Foreign key constraints
    ALTER TABLE messages ADD FOREIGN KEY (sender_id) REFERENCES users(id);
    ALTER TABLE messages ADD FOREIGN KEY (chat_id) REFERENCES chat(id);
    ALTER TABLE message_status ADD FOREIGN KEY (message_id) REFERENCES messages(id);
    ALTER TABLE message_status ADD FOREIGN KEY (receiver_id) REFERENCES users(id);
    ALTER TABLE chat_users ADD FOREIGN KEY (chat_id) REFERENCES chat(id);
    ALTER TABLE chat_users ADD FOREIGN KEY (user_id) REFERENCES users(id);


    -- Insert data into users table
    INSERT INTO users (name) VALUES ('User_1'), ('User_2'), ('User_3'), ('User_4');

    -- Insert data into chat table
    INSERT INTO chat (name, is_group) VALUES ('Private Chat', FALSE), ('Group Chat', TRUE);

    -- Insert data into chat_users table
    INSERT INTO chat_users (chat_id, user_id) VALUES (1, 1), (1, 2), (2, 1), (2, 2), (2, 3), (2, 4);

    -- -- Search for groups that a user is a member of, then show the names and id of the groups
    -- SELECT chat.id, chat.name
    -- FROM chat
    -- JOIN chat_users ON chat.id = chat_users.chat_id
    -- WHERE chat_users.user_id = 1;

    -- Insert data into messages table
    INSERT INTO messages (sender_id, chat_id, message) VALUES (1, 1, 'Hello, how are you?'), (2, 1, 'I am good, thank you!'), (1, 2, 'Hello everyone!'), (2, 2, 'Hi!'), (3, 2, 'Hey there!'), (4, 2, 'Hello!');

    -- SEARCH for messages in a private chat between two users, then show the message and the sender's name with time and the total amount of messages
    SELECT messages.message, users.name AS sender_name, messages.created_at FROM messages JOIN users ON messages.sender_id = users.id WHERE messages.chat_id = ?;


