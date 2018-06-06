import React, { Component } from 'react';
import './App.css';
import { QuickStat } from './QuickStat.js';
import { Plot } from './TimeSeries.js';
import { FieldGroup } from './FormUtils.js';
import { AdminUserMod, AdminModBeacon, AdminModEdge } from './AdminScreens.js';
import ExportScreen from './ExportScreen';
import './bootstrap/css/bootstrap.min.css';
import './bootstrap/css/bootstrap-theme.min.css';
import { decorate, observable } from "mobx";
import { observer } from "mobx-react";
import * as cfg from "./config.js";

import { Grid, Row, Col, Navbar, Nav, NavItem, NavDropdown,
  MenuItem, Button, Alert } from 'react-bootstrap';

class Home extends Component {
  //constructor(props, context) {
  //  super(props, context);
  //}

  render() {
    if (!this.props.loginData.loggedin) {
      return (
        <Row> <Col md={4}>
          <p> Welcome to Beaconpi, You're not logged in so we can't
          show you anything. Click login on the top in order to see some data.
          </p>
        </Col> </Row>
      )
    }
    return (
      <div>
      <Row>
        <Col md={4}>
          <p>Welcome home {this.props.loginData.displayName}, here is the system
          status</p>
        </Col>
      </Row>
      <QuickStat/>
      </div>
    )
  }
}

function modalError(etext) {
  console.log(etext);
}

function updateLogin(loginData) {
  fetch(cfg.app + "/auth/user", {
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
    credentials: 'include',
  }).then((r) => r.json())
  .then((rj) => {
    if ('Error' in rj) {
      modalError(rj.Error);
      loginData.displayName = "";
      loginData.email = "";
      loginData.loggedin = false;
      //TODO(mae) we should perhaps force logout
    }
    if ('Success' in rj) {
      loginData.displayName = rj.DisplayName;
      loginData.email = rj.Email;
      loginData.loggedin = true;
    }
  })
  .catch((error) => {
      modalError("Error receiving confirmation from server");
  });
}

function doLogout(loginData) {
  fetch(cfg.app + "/auth/logout", {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify({
    }),
  }).then((r) => r.json())
  .then((rj) => {
    if ('Error' in rj) {
      modalError(rj.Error);
      loginData.displayName = "";
      loginData.email = "";
      loginData.loggedin = false;
    }
    if ('Success' in rj) {
      loginData.displayName = "";
      loginData.email = "";
      loginData.loggedin = false;
    }
  })
  .catch((error) => {
      modalError("Error receiving confirmation from server");
  });
}

class LoginData {
  //TODO(mae) is this safe?
  id = Math.random();
  loggedin = false;
  displayName = "";
  email = "";
}
decorate(LoginData, {
  loggedin: observable,
  displayName: observable,
  email: observable
})

var Login = observer(
class Login extends Component {
  
  constructor(props, context) {
    super(props, context);

    this.handleChangeEmail = this.handleChangeEmail.bind(this)
    this.handleChangePassword = this.handleChangePassword.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)

    this.state = {
      valueEmail: '',
      valuePassword: '',
      submitted: false,
      errortext: "",
    };
  }

  handleChangeEmail(e) {
    this.setState({ valueEmail: e.target.value });
  }
  handleChangePassword(e) {
    this.setState({ valuePassword: e.target.value });
  }
  handleSubmit(e) {
    this.setState({submitted: true});
    var that = this;
    fetch(cfg.app + "/auth/login", {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({
        "Email": this.state.valueEmail,
        "Passphrase": this.state.valuePassword
      }),
    }).then((r) => r.json())
    .then((rj) => {
      if ('Error' in rj) {
        that.setState({
          errortext: rj.Error
        });
      }
      if ('Success' in rj) {
        that.props.loginData.displayName = rj.DisplayName;
        that.props.loginData.email = that.state.valueEmail;
        that.props.loginData.loggedin = true;
      }
      that.setState({
        valuePassword: '',
        submitted: false
      });
    })
    .catch((error) => {
      that.setState({
        valuePassword: '',
        submitted: false,
        errortext: "Error receiving confirmation from server"
      });
    });
  }

  render() {
    if (this.props.loginData.loggedin) {
      return (
        <h3> You are logged in </h3>
      )
    } else {
      var enableSubmit = this.state.valueEmail.length > 2 && 
        this.state.valuePassword.length > 6;
      return (
        <Col xs={12} sm={6}>
          <form>
            <FieldGroup
              id="formControlEmail" type="email"
              label="Email address" placeholder="Enter email"
              onChange={this.handleChangeEmail}
              disabled={this.state.submitted}
            />
            <FieldGroup
              id="formControlPassphrase" type="password"
              label="Passphrase" placeholder="Enter Passphrase"
              value={this.state.valuePassword}
              onChange={this.handleChangePassword}
              disabled={this.state.submitted}
            />
            <Button bsStyle="success"
              type="submit"
              disabled={!enableSubmit || this.state.submitted}
              onClick={this.handleSubmit}
            >{this.state.submitted ? "Checking" : "Login"}</Button>
          </form>
          {this.state.errortext !== "" && 
            <Alert bsStyle="danger">{this.state.errortext}</Alert>}
        </Col>
      )
    }
  }
})

var App = observer(
class App extends Component {
  constructor(props, context) {
    super(props, context);
    this.handleLogout = this.handleLogout.bind(this);
    this.handleNav = this.handleNav.bind(this);
    var landing = "home";
    if (window.location.hash) {
      landing = window.location.hash.substring(1).toLowerCase();
    }
    this.state = {
      view: landing,
    };
    updateLogin(loginData)
  }

  handleLogout(e) {
  }

  handleNav(eid) {
    switch (eid) {
      case 1: this.setState({view: "home"}); break;
      case 2: this.setState({view: "plot"}); break;
      case 3: this.setState({view: "export"}); break;
      case 20.1: this.setState({view: "usermod"}); break;
      case 20.2: this.setState({view: "modbeacon"}); break;
      case 20.3: this.setState({view: "modedge"}); break;
      case 50: this.setState({view: "login"}); break;
      case 51: doLogout(loginData); break;
      default:
    }
  }

  render() {
    var view;
    switch (this.state.view) {
      case "login": view = <Login loginData={loginData} />; break;
      case "home": view = <Home loginData={loginData} />; break;
      case "plot": view = <Plot />; break;
      case "usermod": view = <AdminUserMod />; break;
      case "modbeacon": view = <AdminModBeacon />; break;
      case "modedge": view = <AdminModEdge />; break;
      case "export": view = <ExportScreen />; break;
      default:
    }

    var loginopt = <NavItem eventKey={50} href="#Login">Login</NavItem>;
    if (loginData.loggedin) {
      loginopt = <NavItem eventKey={51} href="#">Logout</NavItem>
    }

    return (
      <div className="App">
        <Navbar>
          <Navbar.Header>
            <Navbar.Brand>
              Beaconpi
            </Navbar.Brand>
          </Navbar.Header>
          <Nav onSelect={this.handleNav}>
            <NavItem eventKey={1} href="#Home">Home</NavItem>
          </Nav>
            {loginData.loggedin && <Nav onSelect={this.handleNav}>
              <NavItem eventKey={2} href="#Plot">Plot</NavItem>
              <NavItem eventKey={3} href="#Lateration">Lateration</NavItem>
              <NavItem eventKey={4} href="#Export">Export</NavItem>
              <NavDropdown eventKey={20} title="Admin" id="basic-nav-dropdown">
                <MenuItem href="#UserMod" eventKey={20.1}>Modify Users</MenuItem>
                <MenuItem href="#ModBeacon" eventKey={20.2}>Modify Beacons</MenuItem>
                <MenuItem href="#ModEdge" eventKey={20.3}>Modify Edges</MenuItem>
              </NavDropdown>
            </Nav>}
          <Nav pullRight onSelect={this.handleNav}>
            {loginopt}
          </Nav>
        </Navbar>
        <Grid>
          {view}
        </Grid>
      </div>
    );
  }
})

var loginData = new LoginData();

export default App;
