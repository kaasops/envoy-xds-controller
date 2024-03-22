import errorInfo from '../../assets/errors/error.gif';

function ErrorMessage(){
    return (
        <img src={errorInfo}
             alt='Error'
             style={
                 {
                     display: "block",
                     width: '250px',
                     height: '250px',
                     objectFit: 'contain',
                     margin: '0 auto'
                 }
             }
        />
    )
}

export default ErrorMessage;