
import React from "react";
import { withStyles } from '@material-ui/core/styles';
import SignIn from "../components/SignIn/SignIn";

const styles = theme => ({
    root: {
      flexGrow: 1,
      zIndex: 1,
      overflow: 'hidden',
      position: 'relative',
      display: 'flex',
      height: '100%',
    },

    content: {
      flexGrow: 1,
      backgroundColor: theme.palette.background.default,
      padding: theme.spacing.unit * 3,
      minWidth: 0, // So the Typography noWrap works
    },
});

class SigninView extends React.Component {

    constructor(props){
        super(props);
        this.classes = this.props.classes;
    };
    
    render() {
        return (
        <div >

            <div className={this.classes.root}>
                <SignIn />
            </div>
        </div>
    )}
}

export default withStyles(styles)(SigninView);
