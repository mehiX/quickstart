// Imports
import React, { Component } from "react";
import styles from "./index.module.scss";
import firebase, { auth, provider } from "../../firebase.js";
import {
  getAuth,
  signInWithPopup,
  GoogleAuthProvider,
  signOut,
} from "firebase/auth";

// Requires
require("firebase/auth");

// Set interfaces
interface props {
  onHandleUser: (data: data) => void;
  onHandleError: (error: string) => void;
}

interface data {
  user: object | null
  userID: string | null;
  displayName: string | null;
  email: string | null;
}

class Login extends Component<props, data> {
  constructor(props: any) {
    super(props);
    this.state = {
      user: null,
      userID: null,
      displayName: null,
      email: null,
    };
    this.login = this.login.bind(this);
  }
  login() {
    signInWithPopup(auth, provider)
      .then((result) => {
        // This gives you a Google Access Token. Can be used to access the Google API.
        const credential = GoogleAuthProvider.credentialFromResult(result);
        const token = credential ? credential.accessToken : null;
        // The signed-in user info.
        const user = result.user;
        const userID = user.uid;
        const displayName = user.displayName;
        const email = user.email;
        // Set state
        this.setState({
          user,
          userID,
          displayName,
          email,
        });
        this.props.onHandleUser({user, userID, displayName, email });
        this.props.onHandleError('');
      })
      .catch((error) => {
        // Handle Errors here.
        const errorCode = error.code;
        const errorMessage = error.message;
        // The email of the user's account used.
        const email = error.email;
        // The AuthCredential type that was used.
        const credential = GoogleAuthProvider.credentialFromError(error);
        this.props.onHandleError(errorMessage);
      });
  }
  render() {
    return (
      <div className={styles.loginWrapper}>
        <h1>LOGIN</h1>
        <p>Log in with your Google account.</p>
        <button className={styles.logBtn} onClick={this.login}>
          Log In
        </button>
      </div>
    );
  }
}

export default Login;
