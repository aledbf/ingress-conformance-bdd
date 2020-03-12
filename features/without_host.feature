        @sig-network @conformance @release-1.19 @feature-without-host
Feature: Ingress without host field
  I want to expose Services using Ingress definitions wihout using host field

        Scenario: Simple ingress without host
            Given a new random namespace
              And creating objects from directory "scenarios/005"
             When the ingress status shows the IP address or FQDN where is exposed
              And send GET HTTP request
             Then the HTTP response code is 200
              And Header "Host" is not present
