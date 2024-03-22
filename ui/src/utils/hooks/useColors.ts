import { useTheme } from "@mui/material";
import tokens from "../../theme/colors";

const useColors = () => {
    const theme = useTheme();
    const colors = tokens(theme.palette.mode);

    return { theme, colors }
}

export default useColors;