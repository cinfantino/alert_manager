import React from 'react'
import { AuthConsumer } from './SessionContext'
import { Link } from 'react-router-dom'

export default () => (
    <header>
      <AuthConsumer>
        {({ isAuth, login, logout }) => (
          <div>
            <h3>
              <Link to="/">
                HOME
              </Link>
            </h3>
            {isAuth ? (
              <ul>
                <Link to="/dashboard">
                  Dashboard
                </Link>
                <button onClick={logout}>
                  logout
                </button>
              </ul>
            ) : (
              <button onClick={login}>
                login
            </button>
            )}
          </div>
        )}
      </AuthConsumer>
    </header>
  )