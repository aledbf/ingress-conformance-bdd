package io.k8s;

import java.io.File;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

import net.masterthought.cucumber.Configuration;
import net.masterthought.cucumber.ReportBuilder;
import net.masterthought.cucumber.presentation.PresentationMode;
import net.masterthought.cucumber.sorting.SortingMethod;

public class Main {

    public static void main(String[] args) throws IOException {
        String outputDirectory = System.getenv("OUTPUT_DIRECTORY");
        if (outputDirectory == null) {
            outputDirectory = "/report-output";
        }
        File reportOutputDirectory = new File(outputDirectory);

        String inputJSON = System.getenv("INPUT_JSON");
        if (inputJSON == null) {
            inputJSON = "/input.json";
        }

        List<String> jsonFiles = new ArrayList<>();
        jsonFiles.add(inputJSON);

        String buildNumber = "1";
        String projectName = "Ingress Conformance Test";

        Configuration configuration = new Configuration(reportOutputDirectory, projectName);
        configuration.setBuildNumber(buildNumber);
        configuration.addClassifications("Release", "1.19");
        configuration.setSortingMethod(SortingMethod.NATURAL);
        configuration.addPresentationModes(PresentationMode.EXPAND_ALL_STEPS);

        String trendJSON = System.getenv("TREND_JSON");
        if (trendJSON == null) {
            trendJSON = "/report-output/trends.json";
        }

        configuration.setTrendsStatsFile(new File(trendJSON));

        ReportBuilder reportBuilder = new ReportBuilder(jsonFiles, configuration);
        reportBuilder.generateReports();
    }
}
