// Imports
import React, { Component } from "react";
import firebase, { auth, provider } from "../../firebase.js";
import { onAuthStateChanged } from "firebase/auth";

// Requires
require("firebase/auth");

// Interfaces
interface props {
  onHandleUser: (data: data) => void;
}

// Set interfaces
interface data {
  user: object | null;
  userID: string | null;
  displayName: string | null;
  email: string | null;
}
interface props {
  onHandleUser: (data: data) => void,
}

class Auth extends Component<props, data> {
  constructor(props: any) {
    super(props);
    this.state = {
      user: null,
      userID: null,
      displayName: null,
      email: null,
    };
  }

  componentDidMount() {
    onAuthStateChanged(auth, (user) => {
      if (user) {
        console.log('fired');
        // User is signed in, see docs for a list of available properties
        // https://firebase.google.com/docs/reference/js/firebase.User
        // The signed-in user info.
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
      } else {
        // User is signed out
      }
    });
  }

  render() {
    return <></>;
  }
}

export default Auth;
