import React, { Component } from 'react';
import logo from '../../../assets/images/kelp-logo.svg';
import grid from '../../_styles/grid.module.scss';
import styles from './Header.module.scss';

class Header extends Component {
  render() {
    return (
      <header className={styles.header}>
        <div className={grid.container}>
          <div className={styles.headerWrapper}>
            <div className={styles.logoWrapper}>
              <img src={logo} className={styles.logo} alt="Kelp logo" />
              <span className={styles.version}>{this.props.version}</span>
            </div>
          </div>
        </div>
      </header>
    );
  }
}

export default Header;