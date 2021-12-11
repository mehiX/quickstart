// Import the functions from from the SDKs
import { initializeApp } from "firebase/app";
import { getAnalytics } from "firebase/analytics";
import { GoogleAuthProvider, getAuth } from "firebase/auth";

// Web app's Firebase configuration. @TODO: Create project on juuble
const firebaseConfig = {
  apiKey: "AIzaSyCoDcNZZN-jcvbUmad1B8pKkbKVBa-tsbA",
  authDomain: "jublee-a4dd5.firebaseapp.com",
  projectId: "jublee-a4dd5",
  storageBucket: "jublee-a4dd5.appspot.com",
  messagingSenderId: "1002731959941",
  appId: "1:1002731959941:web:ec5d489ec0a1049bc59bec",
  measurementId: "${config.measurementId}"
};

// Initialize Firebase
const firebase = initializeApp(firebaseConfig);

// export needed conts
export const provider = new GoogleAuthProvider();
export const auth = getAuth(firebase);
export const analytics = getAnalytics(firebase);
export default firebase;
