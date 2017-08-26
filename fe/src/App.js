import React, {Component} from "react"
import {Icon, Divider, Container} from "semantic-ui-react"
import "semantic-ui-css/semantic.min.css"


import {Sites} from "./components/Sites"


const Main = () =>
  <Container>
    <h1 style={{paddingTop: '15px'}}><Icon name="lightning"/> Electro</h1>
    <Divider/>
    <Sites/>
  </Container>

class App extends Component {
  render() {
    return (
      <Main/>
    )
  }
}

export default App
