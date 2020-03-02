        @sig-network @conformance @release-1.19
Feature: Default backend
  An Ingress with no rules sends all traffic to a single default backend.
  The default backend is typically a configuration option of the
  Ingress controller and is not specified in your Ingress resources.

  If none of the hosts or paths match the HTTP request in the
  Ingress objects, the traffic is routed to your default backend.

    Rules:
    - Response status code is 404.
    - Response body contains arbitrary text.

        Scenario: Ingress with host and no backend
            Given a new random namespace
              And creating an Ingress with host "foo.bar" without backend serviceName
             Then The error message contains "http.backend.serviceName: Required value"

        Scenario: Ingress with host and invalid backend
            Given a new random namespace
              And an Ingress is created with host "foo.bar" with an invalid backend
             When The ingress status shows the IP address or FQDN where is exposed
              And Header "Host" with value "foo.bar"
              And Send HTTP request with method "GET"
             Then Response status code is 404
              And Header "Host" is "foo.bar"

        Scenario: Ingress should return 404 for paths
            Given a new random namespace
              And an Ingress is created with host "foo.bar" with an invalid backend
             When The ingress status shows the IP address or FQDN where is exposed
              And Header "Host" with value "foo.bar"
             Then Send HTTP request with <path> and <method> checking response status code is 404:
                  |  path   | method  |
                  | /test   | GET     |
                  | /       | POST    |
                  | /       | PUT     |
                  | /       | DELETE  |
                  | /       | GET     |

        Scenario: Ingress with valid host and path /test should return 404 for unmapped path "/"
            Given a new random namespace
              And an Ingress is created with host "foo.bar" with an invalid backend
             When The ingress status shows the IP address or FQDN where is exposed
              And With path "/"
              And Header "Host" with value "foo.bar"
              And Send HTTP request with method "GET"
             Then Response status code is 404
              And Header "Host" is "foo.bar"
