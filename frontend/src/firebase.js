// Import the functions from from the SDKs
import { initializeApp } from "firebase/app";
import { GoogleAuthProvider, getAuth } from "firebase/auth";

// Web app's Firebase configuration. @TODO: Create project on juuble
const firebaseConfig = {
  apiKey: "AIzaSyAOS9UCImq_Qu4QX_NUdIWfVyafpkfvUXE",
  authDomain: "login-demo-f08d6.firebaseapp.com",
  projectId: "login-demo-f08d6",
  storageBucket: "login-demo-f08d6.appspot.com",
  messagingSenderId: "569974589141",
  appId: "1:569974589141:web:058a1648841d4f4b3b045c",
};

// Initialize Firebase
const firebase = initializeApp(firebaseConfig);

// export needed conts
export const provider = new GoogleAuthProvider();
export const auth = getAuth(firebase);
export default firebase;
