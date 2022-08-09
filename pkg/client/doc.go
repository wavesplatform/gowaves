/*
Package client provides HTTP client for waves nodes.

Creating client with default params:

	c, err := client.NewClient()
	...

Client can accept custom node url, http client, and api key:

	c, err := client.NewClient(client.Options{
		Client:  &http.Client{Timeout: 30 * time.Second},
		BaseUrl: "https://nodes.wavesnodes.com",
		ApiKey:  "ApiKey",
	})
	...

Simple example of client usage:

	c, err := client.NewClient()
	if err != nil {
		// handle error
	}
	body, response, err := c.Blocks.First(context.Background())
	if err != nil {
		// handle error
	}
	...
*/
package client
