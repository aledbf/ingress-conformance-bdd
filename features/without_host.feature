        @sig-network @conformance @release-1.19
Feature: Ingress without host field
  I want to expose Services using Ingress definitions wihout using host field

        Scenario: Simple ingress without host
            Given a new random namespace
              And an echo deployment exists
              And an Ingress is created without host using echo service as backend
             When the ingress status shows the IP address or FQDN where is exposed
              And send GET HTTP request
             Then I receive valid HTPP response code 200
              And Header "Host" is not present
