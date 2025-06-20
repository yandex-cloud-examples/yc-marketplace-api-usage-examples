import logging
import os
import sys
from typing import Any
from typing import Union

from flask import Flask
from flask import redirect
from flask import render_template
from flask import request

import config
import handlers

app = Flask(__name__)

port = int(os.getenv("PORT", "8080"))
app.secret_key = config.SECRET_KEY  # Use SECRET_KEY from config

log = logging.getLogger('app')
log.setLevel(logging.DEBUG)
log.addHandler(logging.StreamHandler(stream=sys.stdout))


@app.route('/')
def index() -> str:
    """Renders the main page.

    If the user is logged in and has a product instance ID, renders the app page.
    Otherwise, renders the index page.
    """
    user = handlers.get_user()
    token = request.args.get('token')

    if user and token and not user.product_instance_id:
        # If user is logged in, a token is present, but they are not yet bound,
        # attempt to bind them automatically.
        bind_error = handlers.bind(user.login, token)
        if bind_error:
            # If binding fails, pass the error to the template
            # The template should be able to display this error.
            return render_template('index.jinja', user=user, token=token, error=bind_error)
        # If binding is successful, user object will be updated, redirect to refresh
        return redirect('/')

    if user is not None and user.product_instance_id is not None:
        return render_template('app.jinja', user=user)
    return render_template('index.jinja', user=user, token=token)  # Pass user and token


@app.route('/login', methods=['POST'])
def login_post() -> Union[str, Any]:
    """Handles user login attempts via POST request."""
    error = handlers.login(request.form['login'], request.form['password'])
    token = request.args.get('token')
    if error is None:
        # If login is successful and a token is present, redirect to index
        # which will attempt to auto-bind if not already bound.
        return redirect(f'/?token={token}' if token else '/')
    else:
        return render_template('login.jinja',
                               error=error,
                               token=token)


@app.route('/login', methods=['GET'])
def login_get() -> str:  # Added return type hint
    """Renders the login page."""
    return render_template('login.jinja', token=request.args.get('token'))  # Pass token


@app.route('/register', methods=['POST'])
def register_post() -> Union[str, Any]:  # Added return type hint
    """Handles new user registration attempts via POST request."""
    err = handlers.register(request.form['login'], request.form['password'])
    token = request.args.get('token')  # Get token from query params
    if err is not None:
        return render_template('register.jinja', error=err, token=token)
    # If registration is successful and a token is present, redirect to index
    # which will attempt to auto-bind if not already bound.
    return redirect(f'/?token={token}' if token else '/')


@app.route('/register', methods=['GET'])
def register_get() -> str:  # Added return type hint
    """Renders the registration page."""
    return render_template('register.jinja', token=request.args.get('token'))  # Pass token


@app.route('/logout')
def logout() -> Any:  # Added return type hint (Any for redirect)
    """Logs out the current user and redirects to the index page."""
    handlers.logout_user()  # Use the new logout_user handler
    return redirect('/')


@app.route('/bind', methods=['POST'])
def bind_post() -> Any:  # Added return type hint
    """Binds the logged-in user to a product instance via POST request."""
    user = handlers.get_user()
    token = request.form.get('token')  # Get token from form data

    if not user:
        # Should not happen if UI requires login before allowing bind attempt
        return redirect('/login')

    if not token:
        # Handle missing token - perhaps redirect back with an error message
        return render_template('index.jinja', user=user, error="Binding token is missing.")

    error = handlers.bind(user.login, token)
    if error:
        # Pass error to the template to display it
        return render_template('index.jinja', user=user, token=token, error=error)
    return redirect('/')


@app.route('/report', methods=['POST'])
def report_post() -> Any:  # Added return type hint
    """Reports usage for the current user via POST request."""
    user = handlers.get_user()  # Ensure user is fetched for context, though emulate_work also does this
    if not user:
        return redirect('/login')  # Or an error page

    error = handlers.emulate_work(int(request.form.get('amount', 1)))
    if error:
        # Pass error to the app template to display it
        return render_template('app.jinja', user=user, report_error=error)
    return redirect('/')


if __name__ == "__main__":
    # Create tables if they don't exist (moved from handlers or db to ensure it runs on app start)
    # This is a common pattern but consider if it's best placed here or in a separate init script.
    import db

    db.create_tables()
    app.run(debug=True, port=port, host="0.0.0.0")
