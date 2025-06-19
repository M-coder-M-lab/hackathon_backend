import React, { useEffect, useState } from 'react';
import { fireAuth } from './firebase.ts';
import {
  onAuthStateChanged,
  signInWithPopup,
  signOut,
  GoogleAuthProvider,
} from 'firebase/auth';

function App() {
  const [user, setUser] = useState(null);
  const [posts, setPosts] = useState([]);
  const [newPostContent, setNewPostContent] = useState('');
  const [replyContent, setReplyContent] = useState({});

  useEffect(() => {
    const unsubscribe = onAuthStateChanged(fireAuth, (currentUser) => {
      setUser(currentUser);
    });
    return () => unsubscribe();
  }, []);

  useEffect(() => {
    if (user) {
      fetch('http://localhost:8080/posts')
        .then((res) => res.json())
        .then((data) => setPosts(data))
        .catch((err) => console.error('投稿取得失敗', err));
    }
  }, [user]);

  const loginWithGoogle = async () => {
    try {
      const provider = new GoogleAuthProvider();
      await signInWithPopup(fireAuth, provider);
    } catch (err) {
      console.error('Googleログイン失敗', err);
    }
  };

  const logout = () => {
    signOut(fireAuth);
  };

  const createPost = async () => {
    await fetch('http://localhost:8080/posts/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: user.uid, content: newPostContent }),
    });
    setNewPostContent('');
    reloadPosts();
  };

  const likePost = async (postID) => {
    await fetch('http://localhost:8080/posts/like', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: user.uid, post_id: postID }),
    });
    reloadPosts();
  };

  const sendReply = async (postID) => {
    await fetch('http://localhost:8080/posts/reply', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: user.uid,
        post_id: postID,
        content: replyContent[postID],
      }),
    });
    setReplyContent((prev) => ({ ...prev, [postID]: '' }));
    reloadPosts();
  };

  const reloadPosts = () => {
    fetch('http://localhost:8080/posts')
      .then((res) => res.json())
      .then((data) => setPosts(data));
  };

  if (!user) {
    return (
      <div>
        <h2>Googleアカウントでログイン</h2>
        <button onClick={loginWithGoogle}>Googleでログイン</button>
      </div>
    );
  }

  return (
    <div>
      <h2>ようこそ, {user.displayName || user.email}</h2>
      <button onClick={logout}>ログアウト</button>

      <h3>新しい投稿</h3>
      <textarea
        value={newPostContent}
        onChange={(e) => setNewPostContent(e.target.value)}
      /><br />
      <button onClick={createPost}>投稿</button>

      <h3>投稿一覧</h3>
      {posts.map((post) => (
        <div
          key={post.id}
          style={{ border: '1px solid gray', margin: '10px', padding: '10px' }}
        >
          <p>{post.content}</p>
          <p>いいね: {post.likes}</p>
          <button onClick={() => likePost(post.id)}>いいね</button>

          <h4>リプライ:</h4>
          <ul>
            {post.replies.map((reply) => (
              <li key={reply.id}>{reply.content}</li>
            ))}
          </ul>
          <input
            type="text"
            placeholder="リプライを入力"
            value={replyContent[post.id] || ''}
            onChange={(e) =>
              setReplyContent({ ...replyContent, [post.id]: e.target.value })
            }
          />
          <button onClick={() => sendReply(post.id)}>送信</button>
        </div>
      ))}
    </div>
  );
}

export default App;
