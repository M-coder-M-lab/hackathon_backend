import React, { useState } from 'react';
import './App.css'; // 必要に応じてCSSを記述

function App() {
  const API_BASE_URL = 'http://localhost:8080/api'; // GoバックエンドのURL

  // フォームの状態管理
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [passwordHash, setPasswordHash] = useState('');

  const [postUserId, setPostUserId] = useState('');
  const [postContent, setPostContent] = useState('');

  const [replyPostId, setReplyPostId] = useState('');
  const [replyUserId, setReplyUserId] = useState('');
  const [replyContent, setReplyContent] = useState('');

  const [likePostId, setLikePostId] = useState('');
  const [likeUserId, setLikeUserId] = useState('');

  const [responseMessage, setResponseMessage] = useState(''); // APIからのレスポンスメッセージ

  const sendRequest = async (url, method, body) => {
    try {
      const res = await fetch(url, {
        method: method,
        headers: {
          'Content-Type': 'application/json',
          'Access-Control-Allow-Origin': 'http://localhost:3000', // バックエンドのCORS設定と合わせる
          'Access-Control-Allow-Credentials': 'true',
        },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(`APIエラー: ${res.status} ${res.statusText} - ${errorText}`);
      }

      const data = await res.json();
      setResponseMessage(`成功: ${JSON.stringify(data, null, 2)}`);
    } catch (error) {
      setResponseMessage(`エラー: ${error.message}`);
      console.error('APIリクエストエラー:', error);
    }
  };

  // --- ユーザー作成ハンドラ ---
  const handleCreateUser = async (e) => {
    e.preventDefault();
    await sendRequest(`${API_BASE_URL}/users`, 'POST', {
      username,
      email,
      password_hash: passwordHash,
    });
    setUsername('');
    setEmail('');
    setPasswordHash('');
  };

  // --- 投稿作成ハンドラ ---
  const handleCreatePost = async (e) => {
    e.preventDefault();
    await sendRequest(`${API_BASE_URL}/posts`, 'POST', {
      user_id: parseInt(postUserId, 10),
      content: postContent,
    });
    setPostUserId('');
    setPostContent('');
  };

  // --- 返信作成ハンドラ ---
  const handleCreateReply = async (e) => {
    e.preventDefault();
    await sendRequest(`${API_BASE_URL}/replies`, 'POST', {
      post_id: parseInt(replyPostId, 10),
      user_id: parseInt(replyUserId, 10),
      content: replyContent,
    });
    setReplyPostId('');
    setReplyUserId('');
    setReplyContent('');
  };

  // --- いいね作成ハンドラ ---
  const handleCreateLike = async (e) => {
    e.preventDefault();
    await sendRequest(`${API_BASE_URL}/likes`, 'POST', {
      post_id: parseInt(likePostId, 10),
      user_id: parseInt(likeUserId, 10),
    });
    setLikePostId('');
    setLikeUserId('');
  };

  return (
    <div className="App">
      <h1>Go Backend & React Frontend Example</h1>

      <div className="form-section">
        <h2>ユーザーを作成</h2>
        <form onSubmit={handleCreateUser}>
          <input
            type="text"
            placeholder="ユーザー名"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
          />
          <input
            type="email"
            placeholder="メールアドレス"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
          <input
            type="password"
            placeholder="パスワードハッシュ"
            value={passwordHash}
            onChange={(e) => setPasswordHash(e.target.value)}
            required
          />
          <button type="submit">ユーザー作成</button>
        </form>
      </div>

      <div className="form-section">
        <h2>投稿を作成</h2>
        <form onSubmit={handleCreatePost}>
          <input
            type="number"
            placeholder="ユーザーID"
            value={postUserId}
            onChange={(e) => setPostUserId(e.target.value)}
            required
          />
          <textarea
            placeholder="投稿内容"
            value={postContent}
            onChange={(e) => setPostContent(e.target.value)}
            required
          ></textarea>
          <button type="submit">投稿作成</button>
        </form>
      </div>

      <div className="form-section">
        <h2>返信を作成</h2>
        <form onSubmit={handleCreateReply}>
          <input
            type="number"
            placeholder="投稿ID"
            value={replyPostId}
            onChange={(e) => setReplyPostId(e.target.value)}
            required
          />
          <input
            type="number"
            placeholder="ユーザーID"
            value={replyUserId}
            onChange={(e) => setReplyUserId(e.target.value)}
            required
          />
          <textarea
            placeholder="返信内容"
            value={replyContent}
            onChange={(e) => setReplyContent(e.target.value)}
            required
          ></textarea>
          <button type="submit">返信作成</button>
        </form>
      </div>

      <div className="form-section">
        <h2>「いいね」を作成</h2>
        <form onSubmit={handleCreateLike}>
          <input
            type="number"
            placeholder="投稿ID"
            value={likePostId}
            onChange={(e) => setLikePostId(e.target.value)}
            required
          />
          <input
            type="number"
            placeholder="ユーザーID"
            value={likeUserId}
            onChange={(e) => setLikeUserId(e.target.value)}
            required
          />
          <button type="submit">「いいね」作成</button>
        </form>
      </div>

      {responseMessage && (
        <div className="response-message">
          <h3>APIレスポンス:</h3>
          <pre>{responseMessage}</pre>
        </div>
      )}
    </div>
  );
}

export default App;
