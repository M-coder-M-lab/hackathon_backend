App.js

import React, { useEffect, useState } from 'react';
import './App.css';
import { getAuth, signInWithEmailAndPassword, createUserWithEmailAndPassword } from 'firebase/auth';
import { initializeApp } from 'firebase/app';

const firebaseConfig = {
  apiKey: "AIzaSyBQ4L3nC0GJtsy1SllH4x3I5yInEfpMyc0",
  authDomain: "hackathon-b05e3.firebaseapp.com",
  projectId: "hackathon-b05e3",
  storageBucket: "hackathon-b05e3.firebasestorage.app",
  messagingSenderId: "293078170583",
  appId: "1:293078170583:web:21275a789ba589f5d62992",
  measurementId: "G-7GCR8BHD46"
};

const app = initializeApp(firebaseConfig);
const auth = getAuth(app);

const API_URL = 'http://localhost:8080/api';

function App() {
  const [user, setUser] = useState(null);
  const [posts, setPosts] = useState([]);
  const [postContent, setPostContent] = useState('');
  const [replyContent, setReplyContent] = useState({});
  const [summary, setSummary] = useState({});

  useEffect(() => {
    const storedUser = JSON.parse(localStorage.getItem('user'));
    if (storedUser) {
      setUser(storedUser);
      fetchPosts();
    }
  }, []);

  const handleLogin = async () => {
    const email = prompt('メールアドレスを入力');
    const password = prompt('パスワードを入力');
    try {
      const result = await signInWithEmailAndPassword(auth, email, password);
      const { uid, displayName } = result.user;
      const username = displayName || email;

      const res = await fetch(`${API_URL}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ uid, email, username })
      });
      const data = await res.json();
      const newUser = { id: data.user_id, uid, email, username };
      setUser(newUser);
      localStorage.setItem('user', JSON.stringify(newUser));
      fetchPosts();
    } catch (error) {
      console.error('ログイン失敗:', error);
      alert('ログインに失敗しました。');
    }
  };

  const handleRegister = async () => {
    const email = prompt('登録するメールアドレスを入力');
    const password = prompt('登録するパスワードを入力');
    try {
      const result = await createUserWithEmailAndPassword(auth, email, password);
      const { uid } = result.user;
      const username = email;

      const res = await fetch(`${API_URL}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ uid, email, username })
      });
      const data = await res.json();
      const newUser = { id: data.user_id, uid, email, username };
      setUser(newUser);
      localStorage.setItem('user', JSON.stringify(newUser));
      fetchPosts();
    } catch (error) {
      console.error('ユーザー登録失敗:', error);
      alert('ユーザー登録に失敗しました。');
    }
  };

  const fetchPosts = async () => {
    const res = await fetch(`${API_URL}/posts`);
    const data = await res.json();
    setPosts(data);
  };

  const handlePost = async () => {
    await fetch(`${API_URL}/posts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: user.id, content: postContent })
    });
    setPostContent('');
    fetchPosts();
  };

  const handleReply = async (postId) => {
    await fetch(`${API_URL}/replies`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ post_id: postId, user_id: user.id, content: replyContent[postId] })
    });
    setReplyContent((prev) => ({ ...prev, [postId]: '' }));
    fetchPosts();
  };

  const handleSummary = async (postId) => {
    const res = await fetch(`${API_URL}/summary/${postId}`);
    const data = await res.json();
    setSummary((prev) => ({ ...prev, [postId]: data.summary }));
  };

  if (!user) return (
    <div>
      <button onClick={handleLogin}>メールアドレスでログイン</button>
      <button onClick={handleRegister}>新規ユーザー登録</button>
    </div>
  );

  return (
    <div className="App">
      <h1>ようこそ, {user.username}さん</h1>
      <div>
        <textarea
          value={postContent}
          onChange={(e) => setPostContent(e.target.value)}
          placeholder="新しい投稿..."
        ></textarea>
        <button onClick={handlePost}>投稿</button>
      </div>

      <div>
        {posts.map((post) => (
          <div key={post.id} style={{ border: '1px solid #ccc', margin: '10px', padding: '10px' }}>
            <p>{post.content}</p>
            <p>いいね: {post.likes}</p>
            <div>
              <strong>リプライ:</strong>
              {post.replies.map((reply) => (
                <p key={reply.id} style={{ marginLeft: '1em' }}>- {reply.content}</p>
              ))}
            </div>
            <textarea
              value={replyContent[post.id] || ''}
              onChange={(e) => setReplyContent({ ...replyContent, [post.id]: e.target.value })}
              placeholder="リプライ..."
            ></textarea>
            <button onClick={() => handleReply(post.id)}>リプライ送信</button>
            <button onClick={() => handleSummary(post.id)}>要約取得</button>
            {summary[post.id] && <p><strong>要約:</strong> {summary[post.id]}</p>}
          </div>
        ))}
      </div>
    </div>
  );
}

export default App;

