import PropTypes from "prop-types";
import React from "react";
import { Button, Input, Message, Modal } from "semantic-ui-react";

export class DeleteSite extends React.Component {
  state = {
    error: "",
    password: ""
  };

  static propTypes = {
    site: PropTypes.object.isRequired,
    afterDelete: PropTypes.func.isRequired
  };

  needsPassword = () => this.props.site.has_delete_password;

  handleDelete = async () => {
    let url = `/v1/sites/${this.props.site.id}`;
    if (this.state.password) {
      url += `/${this.state.password}`;
    }
    const req = await fetch(url, { method: "DELETE" });
    const data = await req.json();
    if (data.error) {
      this.setState({ error: data.error });
    } else {
      this.props.afterDelete();
    }
  };

  render = () =>
    <Modal
      size="tiny"
      trigger={<Button content="Delete" icon="trash" color="red" />}
      style={{ textAlign: "center" }}
    >
      <Modal.Header>
        Deleting static site <strong>{this.props.site.name}</strong>
      </Modal.Header>
      <Modal.Content image>
        <Modal.Description>
          {this.state.error &&
            <Message color="red">
              {this.state.error}
            </Message>}
          <p>
            Please verify you want to delete site{" "}
            <strong>{this.props.site.name}</strong>
          </p>
          {this.needsPassword() &&
            <Input
              placeholder="Deletion password"
              onChange={e => this.setState({ password: e.target.value })}
            />}
          <Button
            color="red"
            disabled={this.needsPassword() && !this.state.password}
            onClick={this.handleDelete}
          >
            Delete
          </Button>
        </Modal.Description>
      </Modal.Content>
    </Modal>;
}
