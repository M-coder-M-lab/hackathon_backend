// Import the functions you need from the SDKs you need
import { initializeApp } from "firebase/app";
import { getAnalytics } from "firebase/analytics";
// TODO: Add SDKs for Firebase products that you want to use
// https://firebase.google.com/docs/web/setup#available-libraries

// Your web app's Firebase configuration
// For Firebase JS SDK v7.20.0 and later, measurementId is optional
const firebaseConfig = {
  apiKey: "AIzaSyBQ4L3nC0GJtsy1SllH4x3I5yInEfpMyc0",
  authDomain: "hackathon-b05e3.firebaseapp.com",
  projectId: "hackathon-b05e3",
  storageBucket: "hackathon-b05e3.firebasestorage.app",
  messagingSenderId: "293078170583",
  appId: "1:293078170583:web:21275a789ba589f5d62992",
  measurementId: "G-7GCR8BHD46"
};

// Initialize Firebase
const app = initializeApp(firebaseConfig);
const analytics = getAnalytics(app);