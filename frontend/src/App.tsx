import React, { useEffect, useContext, useCallback, useState } from "react";
import { auth } from "../src/firebase.js";
import { onAuthStateChanged } from "firebase/auth";

import Login from "./Components/Login";
import Logout from "./Components/Logout";
import Auth from "./Components/Auth";
import PlaidWrapper from "./Components/PlaidWrapper";

import styles from "./App.module.scss";

const App = () => {
  // Set interface
  interface data {
    user: object | null;
    userID: string | null;
    displayName: string | null;
    email: string | null;
  }

  // set var for use states
  const [userInfo, setUserInfo] = useState<data>({
    user: null,
    userID: null,
    displayName: null,
    email: null,
  });
  const [error, setError] = useState("");

  // Handle user information (filled after login)
  const handleUserInformation = function (data: data): void {
    setUserInfo(data);
  };

  // Handle error
  const handleError = function (error: string): void {
    setError(error);
  };

  return (
    <div className={styles.App}>
      <Auth onHandleUser={handleUserInformation} />
      <div>
        {userInfo.userID && (
          <div className={styles.container}>
            <Logout
              onHandleUser={handleUserInformation}
              onHandleError={handleError}
            />
            <p>
              Hi, {userInfo.displayName} ({userInfo.email})
            </p>
            <h1>Welcome to Jubilee!</h1>
            <PlaidWrapper user={userInfo.user} />
          </div>
        )}
        {!userInfo.userID && (
          <div className={`${styles.centerFlex} ${styles.centerColumn}`}>
            <Login
              onHandleUser={handleUserInformation}
              onHandleError={handleError}
            />
            <div className={`${styles.error} ${error ? styles.show : ""}`}>
              {error}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default App;
