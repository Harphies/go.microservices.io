package data_structures

/*
example 1
	var r map[string]interface{}
	_ = json.NewDecoder(info.Body).Decode(&r)
	version := r["version"].(map[string]interface{})
	logger.Info(fmt.Sprintf("Connection Established with AWS OpenSearch Cluster version %v", version["number"]))
*/

/*
example 2
ev := c.Poll(100)
			if ev == nil {
				continue
			}
switch e := ev.(type) {
			case *kafka.Message:
				fmt.Printf("%% Message on %s:\n%s\n",
					e.TopicPartition, string(e.Value))
				if e.Headers != nil {
					fmt.Printf("%% Headers: %v\n", e.Headers)
				}
			case kafka.Error:
*/

/*
example 3
	payload, err := hook.Parse(r, github.ReleaseEvent, github.PullRequestEvent)
		if err != nil {
			if err == github.ErrEventNotFound {
				// ok event wasn't one of the ones asked to be parsed
			}
		}
		switch payload.(type) {

		case github.ReleasePayload:
			release := payload.(github.ReleasePayload)
			// Do whatever you want from here...
			fmt.Printf("%+v", release)

		case github.PullRequestPayload:
			pullRequest := payload.(github.PullRequestPayload)
			// Do whatever you want from here...
			fmt.Printf("%+v", pullRequest)
		}
*/
