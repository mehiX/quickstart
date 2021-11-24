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

// Interfaces
interface props {
  onHandleUser: (data: data) => void;
  onHandleError: (error: string) => void;
}

// set interface
interface data {
  user: object | null,
  userID: string | null;
  displayName: string | null;
  email: string | null;
}

class Logout extends Component<props, data> {
  constructor(props: any) {
    super(props);
    this.state = {
      user: null,
      userID: null,
      displayName: null,
      email: null,
    };
    this.logout = this.logout.bind(this);
  }
  logout() {
    signOut(auth)
      .then(() => {
        // Sign-out successful.
        // Now unset all the variables
        const user = null;
        const userID = null;
        const displayName = null;
        const email = null;
        // Set state
        this.setState({
          user,
          userID,
          displayName,
          email,
        });
        this.props.onHandleUser({ user, userID, displayName, email });
        this.props.onHandleError('');
      })
      .catch((error) => {
        // An error happened.
      });
  }

  render() {
    return (
      <button className={styles.logBtn} onClick={this.logout}>
        Log out
      </button>
    );
  }
}

export default Logout;
