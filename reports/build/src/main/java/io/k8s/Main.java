package io.k8s;

import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.List;
import java.util.stream.Collectors;
import java.util.stream.Stream;

import net.masterthought.cucumber.Configuration;
import net.masterthought.cucumber.ReportBuilder;
import net.masterthought.cucumber.presentation.PresentationMode;
import net.masterthought.cucumber.sorting.SortingMethod;

public class Main {

	public static void main(String[] args) throws IOException {
		String outputDirectory = System.getenv("OUTPUT_DIRECTORY");
		if (outputDirectory == null) {
			outputDirectory = "/reports";
		}
		File reportOutputDirectory = new File(outputDirectory);

		String inputFiles = System.getenv("INPUT_JSON_FILES");
		if (inputFiles == null) {
			throw new RuntimeException("Environment variable INPUT_JSON_FILES is not optional");
		}

		Stream<Path> walk = Files.walk(Paths.get(inputFiles));
		List<String> reports = walk.map(x -> x.toString()).filter(f -> f.endsWith("-report.json"))
				.collect(Collectors.toList());	
		
		String buildNumber = "1";
		String projectName = "Ingress Conformance Test";

		Configuration configuration = new Configuration(reportOutputDirectory, projectName);
		configuration.setBuildNumber(buildNumber);
		configuration.addClassifications("Release", "1.19");
		configuration.setSortingMethod(SortingMethod.NATURAL);
		configuration.addPresentationModes(PresentationMode.EXPAND_ALL_STEPS);

		String trendJSON = System.getenv("TREND_JSON");
		if (trendJSON == null) {
			trendJSON = "/reports/trends.json";
		}

		configuration.setTrendsStatsFile(new File(trendJSON));

		ReportBuilder reportBuilder = new ReportBuilder(reports, configuration);
		reportBuilder.generateReports();
		
		walk.close();
	}
}
