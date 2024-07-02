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
    is_group_message BOOLEAN DEFAULT FALSE,
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
ALTER TABLE message_status ADD FOREIGN KEY (message_id) REFERENCES messages(id);
ALTER TABLE message_status ADD FOREIGN KEY (receiver_id) REFERENCES users(id);
ALTER TABLE chat_users ADD FOREIGN KEY (chat_id) REFERENCES chat(id);
ALTER TABLE chat_users ADD FOREIGN KEY (user_id) REFERENCES users(id);
