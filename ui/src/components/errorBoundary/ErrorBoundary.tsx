// предохранитель
import { Component, ErrorInfo, ReactNode } from "react";
import ErrorMessage from "../errorMessage/ErrorMessage";

interface Props {
    children?: ReactNode;
}

interface State {
    error: boolean;
}

export class ErrorBoundary extends Component<Props, State> {
    state = {
        error: false,
    }

    componentDidCatch(error: Error, errorInfo: ErrorInfo) {
        console.log(error, errorInfo);
        this.setState({ error: true })
    }

    render() {
        if (this.state.error) {
            return <ErrorMessage />
        }

        // мы в предохранитель оборачиваем компонент и если нет ошибки показываем наш вложенный(дочерний) компонент
        return this.props.children;
    }
}

export default ErrorBoundary;