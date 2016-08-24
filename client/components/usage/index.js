import React, { Component } from 'react';
import { connect } from 'react-redux';
import Helmet from 'react-helmet';
import { IndexLink } from 'react-router';
import { usage, todo } from './styles';
import { example, p, link } from '../homepage/styles';
import { setConfig } from '../../actions';

class Usage extends Component {

  /*eslint-disable */
  static onEnter({store, nextState, replaceState, callback}) {
    // { credentials: 'same-origin' } means that we should include any cookies
    // that correspond to the destination.  The default for fetch is to omit
    // cookies.  See:
    //   https://developer.mozilla.org/en-US/docs/Web/API/GlobalFetch/fetch
    //   (section "credentials")
    fetch('/api/v1/conf', { credentials: 'same-origin' }).then((r) => {
      return r.json();
    }).then((conf) => {
      store.dispatch(setConfig(conf));
      callback();
    });
  }
  /*eslint-enable */

  render() {
    return <div className={usage}>
      <Helmet title='Usage' />
      <h2 className={example}>Usage:</h2>
      <div className={p}>
        <span className={todo}>// TODO: write an article</span>
        <pre className={todo}>config:
          {JSON.stringify(this.props.config, null, 2)}</pre>
      </div>
      <br />
      go <IndexLink to='/' className={link}>home</IndexLink>
      <hr />
      <a className={link} href='?login=alice'>Login as alice</a> &nbsp;|&nbsp;
      <a className={link} href='?login=bob'>Login as bob</a> &nbsp;|&nbsp;
      <a className={link} href='?login=-'>Logout</a> <br/>
      (then refresh)
    </div>;
  }

}

export default connect(store => ({ config: store.config }))(Usage);
