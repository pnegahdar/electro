import PropTypes from "prop-types"
import React from "react"
import {Button, Form, Message, TextArea} from "semantic-ui-react"
import _ from "lodash"
import isGitUrl from "is-git-url"
import {ErrorMessage, Validate, ValidateGroup} from "react-validate"

import "./site.css"


const ValidatedInput = (props) =>
  <div></div>


ValidatedInput.propTypes = {
  name: PropTypes.string.isRequired,
  label: PropTypes.string.isRequired,
  placeholder: PropTypes.string.isRequired,
  errorMessage: PropTypes.string.isRequired,
  onChange: PropTypes.func.isRequired,
}


export class AddSite extends React.Component {

  state = {
    form: {},
    errors: '',
    success: '',
    loading: false,
    showLoginForm: false,
    showKeyForm: false,
    showDeletePassForm: false,
  }

  static propTypes = {
    reload: PropTypes.func.isRequired,
    wildCard: PropTypes.string.isRequired
  }

  handleChange = (e, {value, name}) => {
    if (name === "hostnames") {
      value = this.splitAndCorrectHosts(value)
    }
    this.setState((currentState) => {
      return {form: {...currentState.form, ...{[name]: value}}}
    })
  }

  create = async () => {
    this.setState({loading: true, errors: '', success: ''}, async () => {
      const req = await fetch("/v1/sites", {method: "POST", body: JSON.stringify(this.state.form)})
      const resp = await req.json()
      if (resp.error) {
        this.setState({errors: resp.error, loading: false})
      } else {
        const hostnameList = this.state.form.hostnames.join(', ')
        this.setState({
          errors: "", loading: false,
          success: `Site ${this.state.form.name } added with hostnames ${hostnameList} added. Please point CNAMES to ${this.props.wildCard} if not already on the wildcard.`
        })
      }
      this.props.reload()
    })
  }

  validateName = (name) => name.length > 2

  validatePassword = (value) => value.length > 5

  splitAndCorrectHosts = (value) => _.filter(_.map(value.split(/,\s?/), (s) => s.trim().toLowerCase()))

  validateHostnames = (value) => _.every(_.map(this.splitAndCorrectHosts(value), (v) => !/[^a-zA-Z.-]/g.test(v)))

  validateBranch = (value) => value ? !/origin[/]/g.test(value) : true

  validatePath = (value) => value ? !/\.\./g.test(value) : true

  toggleLoginForm = () => {
    this.setState((state) => {
      const newState = !state.showLoginForm
      const form = state.form
      if (!newState) {
        delete form.access_username
        delete form.access_password
      }
      return {showLoginForm: newState, form: form}
    })
  }

  toggleFormField = (switchField, formField) => {
    this.setState((state) => {
      const newState = !state[switchField]
      const form = state.form
      if (!newState) {
        delete form[formField]
      }
      return {[switchField]: newState, form: form}
    })
  }



  render = () => {
    const suggestHostname = this.state.form.name && `${this.state.form.name}.${this.props.wildCard}`.toLowerCase().replace(/ /g, '-')
    return <Form>
      <h2>Add a static site</h2>
      {this.state.errors && <Message color="red">{this.state.errors}</Message>}
      {this.state.success && <Message color="green">{this.state.success}</Message>}
      <ValidateGroup impatientFeedback={true}>
        <Validate impatientFeedback={true} validators={[this.validateName]}>
          <Form.Input name="name" label="Name*" placeholder="My Favorite Project" onChange={this.handleChange}/>
          <ErrorMessage>Name must be at least 3 characters.</ErrorMessage>
        </Validate>
        <Validate impatientFeedback={true} validators={[(value) => isGitUrl(value)]}>
          <Form.Input name="git_repo" label='Git URL*' placeholder='https://github.com/pnegahdar/electro.git' onChange={this.handleChange}/>
          <ErrorMessage>A valid git url (https://projec.git, git@)</ErrorMessage>
        </Validate>
        <Validate impatientFeedback={true} validators={[this.validateHostnames]}>
          <Form.Input name="hostnames" label="Hostnames*" placeholder="apidocs.domain.com, docs.domain.com" onChange={this.handleChange}/>
          {(suggestHostname && <small>Suggested: <span>{suggestHostname}</span></small>) || <div/>}
          <ErrorMessage>A valid list of hostnames</ErrorMessage>
        </Validate>
        <Validate impatientFeedback={true} validators={[this.validateBranch]}>
          <Form.Input name="branch" label='Branch' placeholder='master' onChange={this.handleChange}/>
          <ErrorMessage>Do not include "origin/"</ErrorMessage>
        </Validate>
        <Validate impatientFeedback={true} validators={[this.validatePath]}>
          <Form.Input name="root" label='Static Root' placeholder='/' onChange={this.handleChange}/>
          <ErrorMessage>No parent paths (..)</ErrorMessage>
        </Validate>
        <Button.Group style={{marginBottom: '15px'}}>
          <Button icon="lock" active={this.state.showLoginForm} onClick={this.toggleLoginForm}/>
          <Button icon="key" active={this.state.showKeyForm} onClick={() => this.toggleFormField('showKeyForm', 'ssh_key')}/>
          <Button icon="shield" active={this.state.showDeletePassForm} onClick={() => this.toggleFormField('showDeletePassForm', 'delete_password')}/>
        </Button.Group>
        {!this.state.showLoginForm ? <div></div> :
          <Validate impatientFeedback={true} validators={[this.validateName]}>
            <Form.Input name="access_username" label='Auth Username' placeholder='admin' onChange={this.handleChange}/>
            <ErrorMessage>Must be at least 3 characters long.</ErrorMessage>
          </Validate>
        }
        {!this.state.showLoginForm ? <div></div> :
          <Validate impatientFeedback={true} validators={[this.validatePassword]}>
            <Form.Input name="access_password" label='Auth Password' placeholder='admin' onChange={this.handleChange}/>
            <ErrorMessage>Must be at least 5 characters long.</ErrorMessage>
          </Validate>
        }
        {!this.state.showKeyForm ? <div></div> :
          <Validate impatientFeedback={true} validators={[this.validatePassword]}>
            <Form.Input control={TextArea} name="ssh_key" label='SSH Key' placeholder='--Paste--' onChange={this.handleChange}/>
            <ErrorMessage>Must be at least 5 characters long.</ErrorMessage>
          </Validate>
        }
        {!this.state.showDeletePassForm ? <div></div> :
          <Validate impatientFeedback={true} validators={[this.validatePassword]}>
            <Form.Input name="delete_password" label='Deletion password' placeholder='password' onChange={this.handleChange}/>
            <ErrorMessage>Must be at least 5 characters long.</ErrorMessage>
          </Validate>
        }
        <br/>
        <Button color="blue" type="submit" onClick={this.create} loading={this.state.loading}>Create</Button>
      </ValidateGroup>
    </Form>
  }
}



