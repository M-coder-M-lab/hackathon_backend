DROP TABLE IF EXISTS user;
CREATE TABLE user (
    id char(26) NOT NULL primary key,
    name varchar(50) NOT NULL,
    age int(3) NOT NULL
);
INSERT INTO user (id, name, age) VALUES
('00000000000000000000000001', 'hanako', 20),
('00000000000000000000000002', 'taro', 30);