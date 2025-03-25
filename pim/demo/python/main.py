import logging
import os
import sys

from flask import Flask
from flask import redirect
from flask import render_template
from flask import request
from flask import session

import handlers

app = Flask(__name__)

port = int(os.getenv("PORT", "8080"))
app.secret_key = ']mxJ]%98CsQB(LlHPiCl(*R`F,kAu'

log = logging.getLogger('app')
log.setLevel(logging.DEBUG)
log.addHandler(logging.StreamHandler(stream=sys.stdout))


@app.route('/')
def index():
    log.info(session.get('login', None))
    return render_template('index.jinja', token=request.args.get('token'))


@app.route('/login', methods=['POST'])
def login_post():
    error = handlers.login(request.form['login'], request.form['password'])
    token = request.args.get('token')
    if error is None:
        return redirect(f'/?token={token}' if token is not None else '/')
    else:
        return render_template('login.jinja',
                               error=error,
                               token=token)


@app.route('/login', methods=['GET'])
def login_get():
    return render_template('login.jinja')


@app.route('/register', methods=['POST'])
def register_post():
    err = handlers.register(request.form['login'], request.form['password'])
    if err is not None:
        return render_template('register.jinja', error=err)
    token = request.args.get('token')
    return redirect(f'/?token={token}' if token is not None else '/')


@app.route('/register', methods=['GET'])
def register_get():
    return render_template('register.jinja', token=request.args.get('token'))

@app.route('/logout')
def logout():
    try:
        session.pop('login', None)
    finally:
        return redirect('/')

@app.route('/bind', methods=['POST'])
def bind_post():
    handlers.bind(session['login'], token=request.args.get('token'))
    return redirect('/')

if __name__ == "__main__":
    app.run(debug=True, port=port, host="0.0.0.0")
