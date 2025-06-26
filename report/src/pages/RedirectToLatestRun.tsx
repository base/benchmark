import { Navigate } from "react-router-dom";

const RedirectToLatestRun = () => {
  return <Navigate to={`/latest`} />;
};

export default RedirectToLatestRun;
